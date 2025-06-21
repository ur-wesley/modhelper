# R2ModMan Profile Sharer

A simple desktop app that **automatically installs modding profiles** for games like R.E.P.O. and Lethal Company. One-click download and installation of complete mod setups.

## What This App Does

**🎮 Install Complete Modding Profiles**

- Downloads pre-configured mod collections from the internet
- Installs them directly into r2modman (mod manager)
- Launches your games with all mods working

**🔄 Automatic Updates**

- Detects when new mod versions are available
- Updates your profiles with one click
- Keeps your mods current

**🚀 Easy Game Launching**

- Launches games through Steam with mods enabled
- No manual configuration needed
- Just click "Play" and it works

## How It Works

1. **Search for your game** (e.g., "R.E.P.O.", "Lethal Company")
2. **Click "Install"** to download the mod profile
3. **Click "Play"** to launch the game with mods

The app handles all the technical stuff - downloading mods, installing BepInEx, configuring everything correctly.

## Requirements

- **Windows 10/11**
- **Steam** (for launching games)
- **r2modman** (mod manager - app will help you get it)

## Download & Installation

### Option 1: Download Release (Recommended)

1. Go to [Releases](https://github.com/ur-wesley/modhelper/releases)
2. Download the latest `.exe` file
3. Run it - no installation needed

### Option 2: Build from Source

```bash
git clone https://github.com/ur-wesley/modhelper.git
cd modhelper
go install fyne.io/fyne/v2/cmd/fyne@latest
fyne package -os windows -icon icon.png
```

## Usage

### Basic Usage

1. **Run the app** - `ModHelper.exe`
2. **Find your game** - Type in the search box
3. **Install profile** - Click "Installieren"
4. **Play** - Click "Mit Profil spielen"

### Admin Mode (Advanced)

```bash
ModHelper.exe --admin
```

- Configure custom profile sources
- Change installation directory
- Advanced troubleshooting

## Supported Games

Currently supports games with modding profiles available:

- **R.E.P.O.** - Complete mod collection for enhanced gameplay
- **Lethal Company** - Curated mod packs
- More games can be added via manifest configuration

## Troubleshooting

**Game won't start with mods?**

- Make sure r2modman is installed
- Try running the app as administrator
- Check that Steam is running

**Mods not working?**

- Click "Update Profile" if available
- Restart Steam and try again

**Can't find r2modman?**

- The app will prompt you to download it
- Install r2modman first, then try again

## Technical Details

**What happens when you install a profile:**

1. Downloads a `.r2z` file containing mod information
2. Downloads individual mods from Thunderstore
3. Installs BepInEx (mod framework)
4. Configures everything in r2modman's directory
5. Sets up Steam launch parameters

**File locations:**

- Profiles: `%AppData%\r2modmanPlus-local\[GAME]\profiles\`
- Config: `config.json` (in app directory)

## For Developers

### Building Releases

**Create a new release:**

```powershell
# Test the release process
.\scripts\release.ps1 -Version "v1.0.16" -DryRun

# Create actual release
.\scripts\release.ps1 -Version "v1.0.16"
```

**GitHub Actions automatically:**

- Builds Windows executable using Fyne
- Creates GitHub release
- Names file: `ModHelper.exe`

### Project Structure

```
modhelper/
├── main.go              # Entry point
├── const.go             # App constants
├── ui/user.go           # Main interface
├── internal/
│   ├── config/          # Configuration management
│   ├── profile/         # Profile download/install
│   ├── steam/           # Steam integration
│   └── messages.go      # German text
└── .github/workflows/   # Automated builds
```

## License

This project helps install modding profiles for games. All mods are downloaded from their original sources (Thunderstore, etc.).
