package chatbot

import (
	"encoding/json"
	"fmt"
	"strings"
)

type toolSummaryItem struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"arguments,omitempty"`
}

func ToolSummaryPrompt() string {
	return `You write short, user-friendly summaries of tool calls for an inventory management assistant.

Context about the inventory system:
- Devices have serial numbers, manufacturers, models, statuses, and locations.
- Statuses include values like Available, In Use, Broken, or Storage.
- Locations are physical areas like Storage, Front Desk, or Event Hall.
- The tools can query devices/models, update device status/location, add notes, and fetch inventory stats.

Instructions:
- Summarize the tool calls being made by the assistant
- Format the summary as a verb-first action statement using the present participle (“-ing” form), with no pronouns and no implied actor
- Do NOT describe results or outcomes.
- Output ONLY the sentence, no quotes or extra text.

Examples:
- "Checking how many devices are available in Storage."
- "Updating devices' status to Storage"
- "Updating the status of device SN12345 to In Use."
`
}

func TitleSummaryPrompt() string {
	return `You generate short, user-friendly conversation titles for an inventory management assistant.

Instructions:
- Use the previous title unless the topic has clearly changed.
- Keep titles short (3-7 words).
- Use plain language.
- Output ONLY the title, no quotes or extra text.`
}

func buildToolSummaryInput(calls []ToolCall) string {
	items := make([]toolSummaryItem, len(calls))
	for i, call := range calls {
		var args json.RawMessage
		if call.Function.Arguments != "" {
			args = json.RawMessage(call.Function.Arguments)
		}

		item := toolSummaryItem{
			Name: call.Function.Name,
			Args: args,
		}

		items[i] = item
	}

	data, _ := json.MarshalIndent(items, "", "  ")
	return fmt.Sprintf("Tool calls and results:\n%s", string(data))
}

func buildTitleSummaryInput(previousTitle string, messages []Message) string {
	var sb strings.Builder
	sb.WriteString("Previous title: ")
	if previousTitle == "" {
		sb.WriteString("(none)")
	} else {
		sb.WriteString(previousTitle)
	}
	sb.WriteString("\n\nConversation messages:\n")

	for _, msg := range messages {
		if msg.Content == nil || *msg.Content == "" {
			continue
		}
		sb.WriteString(msg.Role)
		sb.WriteString(": ")
		sb.WriteString(*msg.Content)
		sb.WriteString("\n")
	}

	return sb.String()
}

func BuildToolSummaryInput(calls []ToolCall) string {
	return buildToolSummaryInput(calls)
}

func BuildTitleSummaryInput(previousTitle string, messages []Message) string {
	return buildTitleSummaryInput(previousTitle, messages)
}

func FallbackToolSummary(calls []ToolCall) string {
	if len(calls) == 0 {
		return ""
	}
	var parts []string
	for _, call := range calls {
		switch call.Function.Name {
		case "get_stats":
			parts = append(parts, "Checking inventory stats")
		case "get_locations":
			parts = append(parts, "Listing available locations")
		case "get_statuses":
			parts = append(parts, "Listing valid statuses")
		case "query_devices":
			parts = append(parts, "Searching devices")
		case "get_device":
			parts = append(parts, "Fetching device details")
		case "create_device":
			parts = append(parts, "Creating a new device")
		case "update_device":
			parts = append(parts, "Updating device details")
		case "add_device_note":
			parts = append(parts, "Adding a device note")
		case "query_models":
			parts = append(parts, "Searching device models")
		case "get_model":
			parts = append(parts, "Fetching device model details")
		case "create_model":
			parts = append(parts, "Creating a new device model")
		case "update_model":
			parts = append(parts, "Updating a device model")
		default:
			parts = append(parts, "Running inventory tools")
		}
	}
	return strings.Join(uniqueStrings(parts), " and ")
}

func FallbackTitleSummary(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" || messages[i].Content == nil {
			continue
		}
		title := strings.TrimSpace(*messages[i].Content)
		if title == "" {
			continue
		}
		words := strings.Fields(title)
		if len(words) > 6 {
			words = words[:6]
		}
		return strings.Join(words, " ")
	}
	return ""
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
