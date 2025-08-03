package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
)

const (
	ollamaURL    = "http://localhost:11434/api/generate"
	defaultModel = "gemma3n:e4b"
)

type App struct {
	fyneApp    fyne.App
	window     fyne.Window
	statusText *widget.Label
	resultText *widget.Entry
	modelName  string
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

func main() {
	log.Println("Starting lai app...")
	a := &App{
		modelName: defaultModel,
	}
	log.Println("Setting up app...")
	a.setupApp()
	log.Println("Running app...")
	a.fyneApp.Run()
	log.Println("App finished.")
}

func (a *App) setupApp() {
	a.fyneApp = app.New()
	a.fyneApp.SetIcon(nil) // You can add an icon resource here

	// Create a window that will be hidden by default
	a.window = a.fyneApp.NewWindow("lai")
	a.window.Resize(fyne.NewSize(450, 400))
	a.window.SetCloseIntercept(func() {
		a.window.Hide() // Hide instead of closing
	})

	// Set up keyboard shortcuts
	a.setupKeyboardShortcuts()

	// Create UI elements
	a.statusText = widget.NewLabel("Click 'Translate' to translate text from clipboard")
	a.statusText.Wrapping = fyne.TextWrapWord

	a.resultText = widget.NewMultiLineEntry()
	a.resultText.SetPlaceHolder("Translation will appear here...")
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

	content := container.NewVBox(
		a.statusText,
		widget.NewSeparator(),
		widget.NewLabel("Translation Result:"),
		a.resultText,
		buttonContainer,
	)

	a.window.SetContent(content)
	a.window.Show() // Show initially

	// Try to set up system tray (optional)
	if desk, ok := a.fyneApp.(desktop.App); ok {
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
				a.fyneApp.Quit()
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

	// Get selected text using copy
	text, err := a.getSelectedTextWithCopy()
	if err != nil {
		log.Printf("Error getting selected text: %v", err)
		a.updateStatus("Error: Could not get selected text")
		return
	}

	if strings.TrimSpace(text) == "" {
		a.updateStatus("No text selected")
		return
	}

	a.updateStatus("Translating selected text...")

	// Translate the text
	translation, err := a.translateWithOllama(text)
	if err != nil {
		log.Printf("Translation error: %v", err)
		a.updateStatus(fmt.Sprintf("Translation failed: %v", err))
		return
	}

	a.resultText.SetText(translation)
	a.updateStatus("Translation completed")
}

func (a *App) translateClipboardText() {
	// Get text from clipboard
	text, err := clipboard.ReadAll()
	if err != nil {
		log.Printf("Error reading clipboard: %v", err)
		a.updateStatus("Error: Could not read clipboard")
		return
	}

	if strings.TrimSpace(text) == "" {
		a.updateStatus("No text found in clipboard")
		return
	}

	a.updateStatus("Translating...")

	// Translate the text
	translation, err := a.translateWithOllama(text)
	if err != nil {
		log.Printf("Translation error: %v", err)
		a.updateStatus(fmt.Sprintf("Translation failed: %v", err))
		return
	}

	a.resultText.SetText(translation)
	a.updateStatus("Translation completed")
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

func (a *App) translateWithOllama(text string) (string, error) {
	// Create translation prompt
	prompt := fmt.Sprintf("Translate the following text to English. If it's already in English, translate it to Spanish. Only provide the translation, no explanations:\n\n%s", text)

	reqBody := OllamaRequest{
		Model:  a.modelName,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", ollamaURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return strings.TrimSpace(ollamaResp.Response), nil
}

func (a *App) updateStatus(message string) {
	a.statusText.SetText(message)
}

func (a *App) showSettings() {
	settingsWindow := a.fyneApp.NewWindow("Settings")
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
		a.updateStatus(fmt.Sprintf("Settings saved. Using model: %s", a.modelName))
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
