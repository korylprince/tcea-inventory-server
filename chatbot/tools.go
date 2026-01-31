package chatbot

// GetTools returns all available tool definitions for the AI
func GetTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "query_devices",
				Description: "Search for devices in the inventory. Use this to find devices by serial number, manufacturer, model, status, location, or a general search term. Returns a list of matching devices.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"serial_number": map[string]interface{}{
							"type":        "string",
							"description": "Filter by serial number (partial match)",
						},
						"manufacturer": map[string]interface{}{
							"type":        "string",
							"description": "Filter by device manufacturer (partial match)",
						},
						"model": map[string]interface{}{
							"type":        "string",
							"description": "Filter by device model (partial match)",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Filter by device status (partial match)",
						},
						"location": map[string]interface{}{
							"type":        "string",
							"description": "Filter by device location (partial match)",
						},
						"search": map[string]interface{}{
							"type":        "string",
							"description": "General search term to match across all fields. Use this instead of specific filters when searching broadly.",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_device",
				Description: "Get detailed information about a specific device by its ID, including its event history.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "integer",
							"description": "The device ID",
						},
					},
					"required": []string{"id"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "create_device",
				Description: "Create a new device in the inventory. Requires serial number, model ID, status, and location.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"serial_number": map[string]interface{}{
							"type":        "string",
							"description": "The device serial number (must be unique)",
						},
						"model_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the device model",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "The device status (must be a valid status)",
						},
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The device location (must be a valid location)",
						},
					},
					"required": []string{"serial_number", "model_id", "status", "location"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "update_device",
				Description: "Update a device's status and/or location.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "integer",
							"description": "The device ID",
						},
						"serial_number": map[string]interface{}{
							"type":        "string",
							"description": "New serial number (optional)",
						},
						"model_id": map[string]interface{}{
							"type":        "integer",
							"description": "New model ID (optional)",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "New status (optional)",
						},
						"location": map[string]interface{}{
							"type":        "string",
							"description": "New location (optional)",
						},
					},
					"required": []string{"id"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "add_device_note",
				Description: "Add a note to a device's event history.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device_id": map[string]interface{}{
							"type":        "integer",
							"description": "The device ID",
						},
						"note": map[string]interface{}{
							"type":        "string",
							"description": "The note text to add",
						},
					},
					"required": []string{"device_id", "note"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "query_models",
				Description: "Search for device models by manufacturer and/or model name.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"manufacturer": map[string]interface{}{
							"type":        "string",
							"description": "Filter by manufacturer (partial match)",
						},
						"model": map[string]interface{}{
							"type":        "string",
							"description": "Filter by model name (partial match)",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_model",
				Description: "Get information about a specific device model by its ID.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "integer",
							"description": "The model ID",
						},
					},
					"required": []string{"id"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "create_model",
				Description: "Create a new device model with manufacturer and model name.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"manufacturer": map[string]interface{}{
							"type":        "string",
							"description": "The manufacturer name",
						},
						"model": map[string]interface{}{
							"type":        "string",
							"description": "The model name",
						},
					},
					"required": []string{"manufacturer", "model"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "update_model",
				Description: "Update a device model's manufacturer and/or model name.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "integer",
							"description": "The model ID",
						},
						"manufacturer": map[string]interface{}{
							"type":        "string",
							"description": "New manufacturer name (optional)",
						},
						"model": map[string]interface{}{
							"type":        "string",
							"description": "New model name (optional)",
						},
					},
					"required": []string{"id"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_statuses",
				Description: "Get all valid device statuses that can be used when creating or updating devices.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_locations",
				Description: "Get all valid device locations that can be used when creating or updating devices.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_stats",
				Description: "Get inventory statistics including device counts by location, model, and status, plus recent devices.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
	}
}
