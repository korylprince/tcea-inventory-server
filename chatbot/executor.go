package chatbot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/korylprince/tcea-inventory-server/api"
)

// ToolExecutor dispatches tool calls to API functions
type ToolExecutor struct{}

// NewToolExecutor creates a new tool executor
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{}
}

// Execute runs a tool call and returns the JSON result
func (e *ToolExecutor) Execute(ctx context.Context, name string, arguments string) (string, error) {
	var args map[string]interface{}
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", fmt.Errorf("failed to parse arguments: %w", err)
		}
	}

	var result interface{}
	var err error

	switch name {
	case "query_devices":
		result, err = e.queryDevices(ctx, args)
	case "get_device":
		result, err = e.getDevice(ctx, args)
	case "create_device":
		result, err = e.createDevice(ctx, args)
	case "update_device":
		result, err = e.updateDevice(ctx, args)
	case "add_device_note":
		result, err = e.addDeviceNote(ctx, args)
	case "query_models":
		result, err = e.queryModels(ctx, args)
	case "get_model":
		result, err = e.getModel(ctx, args)
	case "create_model":
		result, err = e.createModel(ctx, args)
	case "update_model":
		result, err = e.updateModel(ctx, args)
	case "get_statuses":
		result, err = e.getStatuses(ctx)
	case "get_locations":
		result, err = e.getLocations(ctx)
	case "get_stats":
		result, err = e.getStats(ctx)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error()), nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(data), nil
}

func getString(args map[string]interface{}, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func getInt64(args map[string]interface{}, key string) int64 {
	if v, ok := args[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func (e *ToolExecutor) queryDevices(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	search := getString(args, "search")
	if search != "" {
		return api.SimpleQueryDevice(ctx, search)
	}
	return api.QueryDevice(ctx,
		getString(args, "serial_number"),
		getString(args, "manufacturer"),
		getString(args, "model"),
		getString(args, "status"),
		getString(args, "location"),
	)
}

func (e *ToolExecutor) getDevice(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id := getInt64(args, "id")
	if id == 0 {
		return nil, fmt.Errorf("id is required")
	}
	device, err := api.ReadDevice(ctx, id, true)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return map[string]string{"error": "device not found"}, nil
	}
	// Also fetch the model
	model, err := device.ReadModel(ctx)
	if err != nil {
		return nil, err
	}
	device.Model = model
	return device, nil
}

func (e *ToolExecutor) createDevice(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	device := &api.Device{
		SerialNumber: getString(args, "serial_number"),
		ModelID:      getInt64(args, "model_id"),
		Status:       api.Status(getString(args, "status")),
		Location:     api.Location(getString(args, "location")),
	}
	id, err := api.CreateDevice(ctx, device)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "message": "device created successfully"}, nil
}

func (e *ToolExecutor) updateDevice(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id := getInt64(args, "id")
	if id == 0 {
		return nil, fmt.Errorf("id is required")
	}

	// Get current device
	device, err := api.ReadDevice(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, fmt.Errorf("device not found")
	}

	// Update fields if provided
	if v := getString(args, "serial_number"); v != "" {
		device.SerialNumber = v
	}
	if v := getInt64(args, "model_id"); v != 0 {
		device.ModelID = v
	}
	if v := getString(args, "status"); v != "" {
		device.Status = api.Status(v)
	}
	if v := getString(args, "location"); v != "" {
		device.Location = api.Location(v)
	}

	if err := api.UpdateDevice(ctx, device); err != nil {
		return nil, err
	}
	return map[string]string{"message": "device updated successfully"}, nil
}

func (e *ToolExecutor) addDeviceNote(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	deviceID := getInt64(args, "device_id")
	note := getString(args, "note")

	if deviceID == 0 {
		return nil, fmt.Errorf("device_id is required")
	}
	if note == "" {
		return nil, fmt.Errorf("note is required")
	}

	eventID, err := api.CreateNoteEvent(ctx, deviceID, api.DeviceEventLocation, note)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"event_id": eventID, "message": "note added successfully"}, nil
}

func (e *ToolExecutor) queryModels(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return api.QueryModel(ctx,
		getString(args, "manufacturer"),
		getString(args, "model"),
	)
}

func (e *ToolExecutor) getModel(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id := getInt64(args, "id")
	if id == 0 {
		return nil, fmt.Errorf("id is required")
	}
	model, err := api.ReadModel(ctx, id)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return map[string]string{"error": "model not found"}, nil
	}
	return model, nil
}

func (e *ToolExecutor) createModel(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	model := &api.Model{
		Manufacturer: getString(args, "manufacturer"),
		Model:        getString(args, "model"),
	}
	id, err := api.CreateModel(ctx, model)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "message": "model created successfully"}, nil
}

func (e *ToolExecutor) updateModel(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id := getInt64(args, "id")
	if id == 0 {
		return nil, fmt.Errorf("id is required")
	}

	// Get current model
	model, err := api.ReadModel(ctx, id)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, fmt.Errorf("model not found")
	}

	// Update fields if provided
	if v := getString(args, "manufacturer"); v != "" {
		model.Manufacturer = v
	}
	if v := getString(args, "model"); v != "" {
		model.Model = v
	}

	if err := api.UpdateModel(ctx, model); err != nil {
		return nil, err
	}
	return map[string]string{"message": "model updated successfully"}, nil
}

func (e *ToolExecutor) getStatuses(ctx context.Context) (interface{}, error) {
	return api.ReadStatuses(ctx)
}

func (e *ToolExecutor) getLocations(ctx context.Context) (interface{}, error) {
	return api.ReadLocations(ctx)
}

func (e *ToolExecutor) getStats(ctx context.Context) (interface{}, error) {
	return api.ReadStats(ctx)
}
