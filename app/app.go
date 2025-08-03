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
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
)

//go:embed lai-60x60@3x.png
var iconData []byte

const (
	ollamaURL    = "http://localhost:11434/api/generate"
	defaultModel = "gemma3n:e4b"
)

type App struct {
	ModelName string

	app         fyne.App
	window      fyne.Window
	statusText  *widget.Label
	resultText  *widget.Entry
	startTime   time.Time
	isStreaming bool
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

func New(app fyne.App) *App {
	return &App{
		app:       app,
		ModelName: defaultModel,
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
	a.statusText = widget.NewLabel("Click 'Translate' to translate text to English")
	a.statusText.Wrapping = fyne.TextWrapWord

	a.resultText = widget.NewMultiLineEntry()
	a.resultText.SetPlaceHolder("English translation will appear here...")
	a.resultText.Wrapping = fyne.TextWrapWord

	// Create buttons
	translateBtn := widget.NewButton("Translate Clipboard", func() {
		a.translateClipboardText()
	})

	translateSelectedBtn := widget.NewButton("Translate Selected", func() {
		a.translateSelectedText()
	})

	closeBtn := widget.NewButton("Hide", func() {
		a.window.Hide()
	})

	settingsBtn := widget.NewButton("Settings", func() {
		a.showSettings()
	})

	buttonContainer := container.NewHBox(translateBtn, translateSelectedBtn, settingsBtn, closeBtn)

	// Create top section with status and buttons
	topSection := container.NewVBox(
		a.statusText,
		widget.NewSeparator(),
		buttonContainer,
		widget.NewSeparator(),
		widget.NewLabel("English Translation:"),
	)

	// Use container.NewBorder to make text box take all remaining space
	content := container.NewBorder(topSection, nil, nil, nil, a.resultText)

	a.window.SetContent(content)
	a.window.Show() // Show initially

	// Try to set up system tray (optional)
	if desk, ok := a.app.(desktop.App); ok {
		menu := fyne.NewMenu("lai",
			fyne.NewMenuItem("Show", func() {
				a.window.Show()
				a.window.RequestFocus()
			}),
			fyne.NewMenuItem("Translate Clipboard", func() {
				a.translateClipboardText()
				a.window.Show()
				a.window.RequestFocus()
			}),
			fyne.NewMenuItem("Translate Selected", func() {
				a.translateSelectedText()
				a.window.Show()
				a.window.RequestFocus()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				a.app.Quit()
			}),
		)
		// System tray setup - this might not work on all macOS versions
		desk.SetSystemTrayMenu(menu)
	}
}

func (a *App) setupKeyboardShortcuts() {
	// Set up global keyboard shortcut for translating selected text
	// Shift+Option+T for translate selected text
	shortcut := &desktop.CustomShortcut{
		KeyName:  fyne.KeyT,
		Modifier: fyne.KeyModifierShift | fyne.KeyModifierAlt,
	}
	a.window.Canvas().AddShortcut(shortcut, func(shortcut fyne.Shortcut) {
		a.translateSelectedText()
		a.window.Show()
		a.window.RequestFocus()
	})

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

func (a *App) translateSelectedText() {
	a.updateStatus("Getting selected text...")
	text, err := a.getSelectedTextWithCopy()
	if err != nil {
		a.updateStatus(fmt.Sprintf("Error: %v", err))
		return
	}

	if text == "" {
		a.updateStatus("No text selected.")
		return
	}

	a.updateStatus("Translating selected text...")
	a.resultText.SetText("") // Clear previous result
	go a.streamTranslateWithOllama(text)
}

func (a *App) translateClipboardText() {
	a.updateStatus("Getting clipboard text...")
	text, err := clipboard.ReadAll()
	if err != nil {
		a.updateStatus(fmt.Sprintf("Error reading clipboard: %v", err))
		return
	}

	if text == "" {
		a.updateStatus("Clipboard is empty.")
		return
	}

	a.updateStatus("Translating clipboard text...")
	a.resultText.SetText("") // Clear previous result
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

func (a *App) streamTranslateWithOllama(text string) {
	a.startTime = time.Now()
	a.isStreaming = true

	// Start timer update goroutine
	go a.updateTimer()

	// Create translation prompt - always translate to English
	prompt := fmt.Sprintf("Translate the following text to English. Only provide the English translation, no explanations or additional text:\n\n%s", text)

	reqBody := OllamaRequest{
		Model:  a.ModelName,
		Prompt: prompt,
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		a.isStreaming = false
		a.updateStatus(fmt.Sprintf("Failed to marshal request: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", ollamaURL, bytes.NewBuffer(jsonData))
	if err != nil {
		a.isStreaming = false
		a.updateStatus(fmt.Sprintf("Failed to create request: %v", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		a.isStreaming = false
		a.updateStatus(fmt.Sprintf("Failed to make request to Ollama: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.isStreaming = false
		a.updateStatus(fmt.Sprintf("Ollama returned status %d", resp.StatusCode))
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
			a.isStreaming = false
			a.updateStatus(fmt.Sprintf("Failed to decode response: %v", err))
			return
		}

		// Append the response chunk
		fullResponse.WriteString(ollamaResp.Response)

		// Update UI with current text
		a.resultText.SetText(fullResponse.String())

		if ollamaResp.Done {
			break
		}
	}

	a.isStreaming = false
	elapsed := time.Since(a.startTime)
	a.updateStatus(fmt.Sprintf("Translation completed in %.1fs", elapsed.Seconds()))
}

func (a *App) updateTimer() {
	for a.isStreaming {
		elapsed := time.Since(a.startTime)
		a.updateStatus(fmt.Sprintf("Translating... %.1fs", elapsed.Seconds()))
		time.Sleep(100 * time.Millisecond)
	}
}

func (a *App) updateStatus(message string) {
	a.statusText.SetText(message)
}

func (a *App) showSettings() {
	settingsWindow := a.app.NewWindow("Settings")
	settingsWindow.Resize(fyne.NewSize(350, 250))

	modelEntry := widget.NewEntry()
	modelEntry.SetText(a.ModelName)
	modelEntry.SetPlaceHolder("Enter Ollama model name")

	ollamaURLEntry := widget.NewEntry()
	ollamaURLEntry.SetText(ollamaURL)
	ollamaURLEntry.SetPlaceHolder("Ollama server URL")

	saveBtn := widget.NewButton("Save", func() {
		// Update the model name
		a.ModelName = modelEntry.Text
		a.updateStatus(fmt.Sprintf("Settings saved. Using model: %s", a.ModelName))
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
