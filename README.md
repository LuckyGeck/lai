<p align="center">
  <img src="app/lai-60x60@3x.png" alt="lai icon" width="90">
</p>

<h1 align="center">lai</h1>

<p align="center">
A Mac menu bar app for translating text from the clipboard using local Ollama models.
</p>

## Features

- **Menu Bar Integration**: Lives in your Mac's menu bar.
- **Local AI Translation**: Uses Ollama for privacy and speed.
- **Clipboard Translation**: Translates clipboard text with one click.
- **Configurable Models**: Choose your preferred Ollama model.
- **Smart Translation**: Translates to English or Spanish automatically.

## Prerequisites

1. **Ollama**: Install and run Ollama locally. [Install Guide](https://ollama.ai/)
   ```bash
   ollama pull gemma3n:e4b
   ```
2. **Go**: Version 1.23.4 or later.

## Getting Started

1. **Clone the repository**
2. **Run the app**:
   ```bash
   go run main.go
   ```

## Usage

- Copy text to your clipboard.
- Click the `lai` icon in the menu bar and select **Translate**.
- To change the model, click **Show** -> **Settings**.

## Building for Distribution

To create a standalone app bundle for macOS:

```bash
# Build for macOS
go build -ldflags="-s -w" -o lai main.go

# Optional: Create an app bundle
mkdir -p lai.app/Contents/MacOS
cp lai lai.app/Contents/MacOS/
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
