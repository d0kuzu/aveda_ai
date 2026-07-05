package constants

import "github.com/sashabaranov/go-openai"

var Tools = []openai.Tool{
	/*
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "calcom_get_available_slots",
				Description: "Get available appointment time slots for a specific date from Cal.com. Call this when the user mentions a day or date they want to visit.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"date": map[string]interface{}{
							"type":        "string",
							"description": "The date to check availability for, in YYYY-MM-DD format (e.g. 2026-05-16)",
						},
					},
					"required": []string{"date"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "calcom_create_booking",
				Description: "Create a booking appointment at a specific time slot on Cal.com. Call this only after the user has selected an available time slot and provided their name and email address.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"start_time": map[string]interface{}{
							"type":        "string",
							"description": "The start time of the appointment in ISO 8601 UTC format (e.g. 2026-05-16T14:00:00Z)",
						},
						"attendee_name": map[string]interface{}{
							"type":        "string",
							"description": "Full name of the person booking the appointment",
						},
						"attendee_email": map[string]interface{}{
							"type":        "string",
							"description": "Email address of the person booking the appointment",
						},
					},
					"required": []string{"start_time", "attendee_name", "attendee_email"},
				},
			},
		},
	*/
	/*
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_available_slots",
				Description: "Get available appointment time slots for a specific date. Call this when the user mentions a day or date they want to visit. Returns a list of available time slots.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"date": map[string]interface{}{
							"type":        "string",
							"description": "The date to check availability for, in YYYY-MM-DD format (e.g. 2026-05-16)",
						},
					},
					"required": []string{"date"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "create_booking",
				Description: "Create a booking appointment at a specific time slot. Call this only after the user has selected an available time slot.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"start_time": map[string]interface{}{
							"type":        "string",
							"description": "The start time of the appointment (e.g. 2026-05-25T11:30:00)",
						},
						"end_time": map[string]interface{}{
							"type":        "string",
							"description": "The end time of the appointment (e.g. 2026-05-25T12:30:00)",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Brief description of the lead",
						},
					},
					"required": []string{"start_time", "end_time", "description"},
				},
			},
		},
	*/
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "send_summary",
			Description: "Send a summary of the conversation (important information about the user, what they were interested in, etc.). Call this function when you give the user a link to a tour.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Important information about the user, their interests, and context of the conversation.",
					},
				},
				"required": []string{"summary"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "mark_grade_11_or_lower",
			Description: "Call this function if the user mentions that they are in Grade 11 or lower (e.g. 'I am in grade 11', 'grade 10'). This marks them as unqualified for certain programs.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "mark_international_student",
			Description: "Call this function when the user confirms they are on a visa or an International Student, specifically in response to the question 'Are you a Canadian citizen, permanent resident, or on a visa?' and they choose the visa/international route. This marks them as unqualified for follow-ups.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "google_calendar_get_slots",
			Description: "Get available time slots for a specific date from Google Calendar. Call this when the user mentions a day or date they want to visit.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"date": map[string]interface{}{
						"type":        "string",
						"description": "The date to check availability for, in YYYY-MM-DD format (e.g. 2026-05-16)",
					},
				},
				"required": []string{"date"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "google_calendar_create_event",
			Description: "Create a calendar event at a specific time (duration is automatically 30 minutes). Call this only after the user has confirmed the exact date and time (e.g. '3 March at 2PM, is it correct?').",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "The title or summary of the event (e.g. 'Campus Tour for [Name]')",
					},
					"start": map[string]interface{}{
						"type":        "string",
						"description": "The start time of the event in RFC3339 format (e.g. 2026-05-25T11:30:00Z)",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Brief description of the lead",
					},
				},
				"required": []string{"title", "start", "description"},
			},
		},
	},
}
