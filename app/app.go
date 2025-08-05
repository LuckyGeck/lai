package app

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
)

//go:embed lai-60x60@3x.png
var iconData []byte

const (
	ollamaURL       = "http://localhost:11434/api/generate"
	ollamaModelsURL = "http://localhost:11434/api/tags"
	defaultModel    = "gemma3n:e4b"
)

type App struct {
	modelName     string
	app           fyne.App
	window        fyne.Window
	modelDropdown *widget.Select

	input  binding.String
	result binding.String
	status binding.String
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type OllamaModel struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
}

type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

func New(app fyne.App) *App {
	return &App{
		app:       app,
		modelName: defaultModel,
	}
}

func (a *App) Run() {
	log.Println("Setting up app...")
	a.setupApp()
	log.Println("Running app...")
	a.app.Run()
	log.Println("App finished.")
}

func (a *App) setupApp() {
	// Set the app icon using embedded PNG
	iconResource := fyne.NewStaticResource("lai-60x60@3x.png", iconData)
	a.app.SetIcon(iconResource)

	// Create a window that will be hidden by default
	a.window = a.app.NewWindow("lai")
	a.window.Resize(fyne.NewSize(600, 500))
	a.window.SetCloseIntercept(func() {
		a.window.Hide() // Hide instead of closing
	})

	// Set up keyboard shortcuts
	a.setupKeyboardShortcuts()

	// Create UI elements
	a.status = binding.NewString()
	a.status.Set("Click 'Translate' to start translating")
	statusText := widget.NewLabelWithData(a.status)
	statusText.Wrapping = fyne.TextWrapWord

	// Create model dropdown
	a.modelDropdown = widget.NewSelect([]string{a.modelName}, func(selected string) {
		if selected != "" {
			a.modelName = selected
			a.setStatus("Model changed to: %s", selected)
		}
	})
	a.modelDropdown.SetSelected(a.modelName)
	a.modelDropdown.PlaceHolder = "Select model..."

	// Model selection container
	modelContainer := container.NewHBox(
		widget.NewLabel("Model:"),
		a.modelDropdown,
		widget.NewButton("Refresh", func() { a.refreshModelDropdown() }),
	)

	a.input = binding.NewString()
	inputText := widget.NewEntryWithData(a.input)
	inputText.SetPlaceHolder("Enter text to translate...")
	inputText.MultiLine = true
	inputText.Wrapping = fyne.TextWrapWord

	a.result = binding.NewString()
	resultText := widget.NewEntryWithData(a.result)
	resultText.SetPlaceHolder("Translation will appear here...")
	resultText.MultiLine = true
	resultText.Wrapping = fyne.TextWrapWord

	buttonContainer := container.NewHBox(
		widget.NewButton("Translate Clipboard", func() { a.translateClipboardText() }),
		widget.NewButton("Translate", func() { a.translateInputText() }),
		widget.NewButton("Settings", func() { a.showSettings() }),
		widget.NewButton("Hide", func() { a.window.Hide() }),
	)

	topSection := container.NewVBox(
		statusText,
		widget.NewSeparator(),
		modelContainer,
		widget.NewSeparator(),
		buttonContainer,
		widget.NewSeparator(),
		widget.NewLabel("Input:"),
		inputText,
		widget.NewLabel("Translation:"),
	)

	content := container.NewBorder(topSection, nil, nil, nil, resultText)

	a.window.SetContent(content)
	a.window.Show()

	// Load available models on startup
	a.refreshModelDropdown()

	if desk, ok := a.app.(desktop.App); ok {
		desk.SetSystemTrayMenu(fyne.NewMenu("lai",
			fyne.NewMenuItem("Show", func() {
				a.window.Show()
				a.window.RequestFocus()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				a.app.Quit()
			}),
		))
	}
}

func (a *App) setupKeyboardShortcuts() {
	// Shift+Option+C for translate clipboard
	clipboardShortcut := &desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierShift | fyne.KeyModifierAlt,
	}
	a.window.Canvas().AddShortcut(clipboardShortcut, func(shortcut fyne.Shortcut) {
		a.translateClipboardText()
		a.window.Show()
		a.window.RequestFocus()
	})
}

func (a *App) translateInputText() {
	a.setStatus("Translating input text...")
	a.result.Set("")
	text, err := a.input.Get()
	if err != nil {
		a.setStatus("Error getting input text: %v", err)
		return
	}
	go a.streamTranslateWithOllama(text)
}

func (a *App) translateClipboardText() {
	a.setStatus("Getting clipboard text...")
	text, err := clipboard.ReadAll()
	if err != nil {
		a.setStatus("Error reading clipboard: %v", err)
		return
	}

	if text == "" {
		a.setStatus("Clipboard is empty.")
		return
	}

	a.setStatus("Translating clipboard text...")
	a.result.Set("")
	a.input.Set(text)
	go a.streamTranslateWithOllama(text)
}

func (a *App) getSelectedTextWithCopy() (string, error) {
	// Save current clipboard content
	originalClip, _ := clipboard.ReadAll()

	// Clear clipboard
	clipboard.WriteAll("")

	// Simulate Cmd+C to copy selected text
	cmd := exec.Command("osascript", "-e", `
		tell application "System Events"
			keystroke "c" using command down
		end tell
	`)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to copy text: %w", err)
	}

	// Wait a bit for clipboard to update
	time.Sleep(100 * time.Millisecond)

	// Get the copied text
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to read clipboard: %w", err)
	}

	// Restore original clipboard content
	if originalClip != "" {
		clipboard.WriteAll(originalClip)
	}

	return text, nil
}

//go:embed prompt.txt
var promptTemplate string

func (a *App) streamTranslateWithOllama(text string) {
	stopTick := make(chan struct{})
	defer close(stopTick)
	go func() {
		ticker := time.Tick(100 * time.Millisecond)
		startTime := time.Now()
		for {
			select {
			case <-stopTick:
				return
			case now := <-ticker:
				a.setStatus("Translating... %.1fs", now.Sub(startTime).Seconds())
			}
		}
	}()

	// Create a smart translation prompt
	prompt := fmt.Sprintf(promptTemplate, text)

	reqBody := OllamaRequest{
		Model:  a.modelName,
		Prompt: prompt,
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		a.setStatus("Failed to marshal request: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", ollamaURL, bytes.NewBuffer(jsonData))
	if err != nil {
		a.setStatus("Failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		a.setStatus("Failed to make request to Ollama: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.setStatus("Ollama returned status %d", resp.StatusCode)
		return
	}

	// Stream the response
	decoder := json.NewDecoder(resp.Body)
	var fullResponse strings.Builder

	for {
		var ollamaResp OllamaResponse
		if err := decoder.Decode(&ollamaResp); err != nil {
			if err.Error() == "EOF" {
				break
			}
			a.setStatus("Failed to decode response: %v", err)
			return
		}

		// Append the response chunk
		fullResponse.WriteString(ollamaResp.Response)

		// Update UI with current text
		a.result.Set(fullResponse.String())

		if ollamaResp.Done {
			break
		}
	}
}

func (a *App) setStatus(format string, args ...any) {
	a.status.Set(fmt.Sprintf(format, args...))
}

func (a *App) fetchAvailableModels() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ollamaModelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var modelsResp OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var modelNames []string
	for _, model := range modelsResp.Models {
		modelNames = append(modelNames, model.Name)
	}

	return modelNames, nil
}

func (a *App) refreshModelDropdown() {
	go func() {
		a.setStatus("Loading available models...")
		models, err := a.fetchAvailableModels()
		if err != nil {
			a.setStatus("Failed to load models: %v", err)
			// Fallback to current model if fetch fails
			if a.modelDropdown != nil {
				a.modelDropdown.Options = []string{a.modelName}
				a.modelDropdown.SetSelected(a.modelName)
				a.modelDropdown.Refresh()
			}
			return
		}

		if len(models) == 0 {
			a.setStatus("No models found on Ollama server")
			return
		}

		// Update dropdown options
		if a.modelDropdown != nil {
			a.modelDropdown.Options = models
			// Select current model if it exists in the list, otherwise select first
			found := false
			for _, model := range models {
				if model == a.modelName {
					a.modelDropdown.SetSelected(a.modelName)
					found = true
					break
				}
			}
			if !found && len(models) > 0 {
				a.modelName = models[0]
				a.modelDropdown.SetSelected(models[0])
			}
			a.modelDropdown.Refresh()
		}

		a.setStatus("Loaded %d models from Ollama server", len(models))
	}()
}

func (a *App) showSettings() {
	settingsWindow := a.app.NewWindow("Settings")
	settingsWindow.Resize(fyne.NewSize(350, 250))

	modelEntry := widget.NewEntry()
	modelEntry.SetText(a.modelName)
	modelEntry.SetPlaceHolder("Enter Ollama model name")

	ollamaURLEntry := widget.NewEntry()
	ollamaURLEntry.SetText(ollamaURL)
	ollamaURLEntry.SetPlaceHolder("Ollama server URL")

	saveBtn := widget.NewButton("Save", func() {
		// Update the model name
		a.modelName = modelEntry.Text
		a.setStatus("Settings saved. Using model: %s", a.modelName)
		settingsWindow.Close()
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		settingsWindow.Close()
	})

	buttonContainer := container.NewHBox(saveBtn, cancelBtn)

	content := container.NewVBox(
		widget.NewLabel("Ollama Model:"),
		modelEntry,
		widget.NewLabel("Ollama URL:"),
		ollamaURLEntry,
		widget.NewLabel("Note: URL changes require app restart"),
		buttonContainer,
	)

	settingsWindow.SetContent(content)
	settingsWindow.Show()
}
