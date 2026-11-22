# Database RPC contract

Edit `database-service/proto/database.proto` and regenerate the Go server and client bindings together. The checked-in bindings use protoc 25.2 and the following Go plugins.

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
```

Add the Go binaries directory to `PATH`. From the repository root, run the following in PowerShell.

```powershell
./database-service/proto/generate.ps1
```

The script generates the bindings in place and copies the contract and generated files into each consumer's `proto/db` directory.
