"""Copy one service's data from a stopped legacy installation into its empty database."""
import argparse
import os
import subprocess
import sys

TABLES = {
    "auth": {
        "users": "id,email,password_hash,role,created_at,updated_at",
        "refresh_tokens": "id,user_id,token_hash,expires_at,created_at",
    },
    "ai": {
        "assistants": "id,user_id,name,configuration,api_token,telegram_bot_token,type,created_at,updated_at",
        "chats": "id,assistant_id,customer_id,message_count,is_end,started_at,created_at,updated_at",
        "messages": "id,chat_id,role,content,time,created_at,updated_at",
        "twilio_configs": "assistant_id,user_id,twilio_number,account_sid,auth_token,created_at,updated_at",
        "blocked_customers": "user_id",
    },
}

def psql(dsn):
    return ["psql", "-X", "-q", "-w", "-v", "ON_ERROR_STOP=1", "-d", dsn]

def transfer(service, source, target):
    tables = TABLES[service]
    if service == "ai":
        check = """SELECT EXISTS (
            SELECT FROM public.assistants a LEFT JOIN public.users u ON u.id::text = a.user_id::text
            WHERE u.id IS NULL
        )"""
        orphaned = subprocess.check_output(psql(source) + ["-Atc", check]).strip()
        if orphaned == b"t":
            raise ValueError("Resolve assistant owner IDs missing from legacy users before importing AI data.")

    refresh_filter = " WHERE EXISTS (SELECT FROM public.users u WHERE u.id::text = t.user_id::text)"
    if service == "auth":
        missing = """SELECT count(*) FROM public.refresh_tokens t
            WHERE NOT EXISTS (SELECT FROM public.users u WHERE u.id::text = t.user_id::text)"""
        count = int(subprocess.check_output(psql(source) + ["-Atc", missing]))
        if count:
            print(f"Skipping {count} refresh token(s) without a user. These sessions require login.", flush=True)

    # COPY text preserves escaped newlines, backslashes and nulls. The source database is read-only here.
    # The two streams stay binary to avoid platform-dependent encoding and newline conversions.
    env = dict(os.environ, PGCLIENTENCODING="UTF8")
    with subprocess.Popen(psql(target), stdin=subprocess.PIPE, env=env) as dest:
        def sql(statement):
            dest.stdin.write(statement.encode("ascii") + b"\n")
            dest.stdin.flush()

        try:
            sql("BEGIN;")
            sql("SET LOCAL search_path TO public;")
            sql("LOCK TABLE " + ", ".join(tables) + " IN ACCESS EXCLUSIVE MODE;")
            populated = " OR ".join(f"EXISTS (SELECT FROM {name})" for name in tables)
            sql(f"DO $$ BEGIN IF {populated} THEN RAISE EXCEPTION 'Destination must be empty'; END IF; END $$;")
            for table, cols in tables.items():
                sql(f"COPY {table} ({cols}) FROM STDIN;")
                select_cols = cols
                if table == "assistants":
                    select_cols = ("id,user_id,name,COALESCE(configuration,''),COALESCE(api_token,''),"
                                   "COALESCE(telegram_bot_token,''),COALESCE(type,'api'),created_at,updated_at")
                where = refresh_filter if table == "refresh_tokens" else ""
                export = f"COPY (SELECT {select_cols} FROM public.{table} t{where}) TO STDOUT"
                subprocess.run(psql(source) + ["-c", export], stdout=dest.stdin, env=env, check=True)
                sql(r"\.")
            if service == "ai":
                sql("SELECT setval(pg_get_serial_sequence('messages','id'), "
                    "COALESCE(MAX(id),1), MAX(id) IS NOT NULL) FROM messages;")
            sql("COMMIT;")
            dest.stdin.close()
            if dest.wait():
                raise ValueError("Destination rejected the import.")
        except BaseException:
            # Closing the connection before COMMIT rolls back all copied tables.
            if dest.poll() is None:
                dest.terminate()
            raise

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("service", choices=TABLES)
    args = parser.parse_args()
    try:
        transfer(args.service, os.environ["LEGACY_DATABASE_URL"], os.environ["TARGET_DATABASE_URL"])
    except subprocess.CalledProcessError:
        sys.exit("Source query failed. See the PostgreSQL error above.")
    except (KeyError, OSError, ValueError) as err:
        sys.exit(str(err))
    print(f"Imported {args.service} data.")
