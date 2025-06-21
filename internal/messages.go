package internal

type Messages struct {
	WindowTitle string
	SearchGames string
	AdminMode   string

	NotInstalled    string
	InstallAndPlay  string
	PlayWithProfile string
	PlayGame        string
	Installing      string
	GameRunning     string
	GameActive      string
	RunningButton   string
	StopGame        string
	Stopping        string
	UpdateProfile   string
	Updating        string

	Download  string
	Install   string
	Launch    string
	Configure string
	Stop      string
	Update    string

	LoadingGames       string
	NoGamesFound       string
	ProfileInstalled   string
	InstallationFailed string
	LaunchFailed       string
	StopFailed         string
	UpdateFailed       string

	R2ModmanStatus string
	SteamStatus    string
	ManifestStatus string

	ManifestURL string
	TargetDir   string
	Save        string
	Cancel      string

	InfoTitle   string
	InfoContent string
	Close       string

	Error            string
	SteamNotFound    string
	R2ModmanNotFound string
	NetworkError     string
}

func German() Messages {
	return Messages{
		WindowTitle: "R2ModMan Profile Sharer",
		SearchGames: "Spiele suchen...",
		AdminMode:   "Admin-Modus",

		NotInstalled:    "Nicht installiert",
		InstallAndPlay:  "Installieren & Spielen",
		PlayWithProfile: "Mit Profil spielen",
		PlayGame:        "Spiel starten",
		Installing:      "Installiere...",
		GameRunning:     "Läuft gerade",
		GameActive:      "Aktiv",
		RunningButton:   "Läuft...",
		StopGame:        "Spiel beenden",
		Stopping:        "Beende...",
		UpdateProfile:   "Profil aktualisieren",
		Updating:        "Aktualisiere...",

		Download:  "Herunterladen",
		Install:   "Installieren",
		Launch:    "Starten",
		Configure: "Konfigurieren",
		Stop:      "Beenden",
		Update:    "Aktualisieren",

		LoadingGames:       "Lade Spiele...",
		NoGamesFound:       "Keine Spiele gefunden",
		ProfileInstalled:   "Profil erfolgreich installiert",
		InstallationFailed: "Installation fehlgeschlagen",
		LaunchFailed:       "Start fehlgeschlagen",
		StopFailed:         "Beenden fehlgeschlagen",
		UpdateFailed:       "Aktualisierung fehlgeschlagen",

		R2ModmanStatus: "r2modman",
		SteamStatus:    "Steam",
		ManifestStatus: "Manifest",

		ManifestURL: "Manifest-URL:",
		TargetDir:   "Zielordner:",
		Save:        "Speichern",
		Cancel:      "Abbrechen",

		InfoTitle: "Anleitung",
		InfoContent: `VERWENDUNG:

1. SPIEL SUCHEN
   • Tippe Spielnamen in Suchfeld
   • Wähle aus der Liste

2. PROFIL INSTALLIEREN
   • Klicke "Installieren & Spielen"
   • Warte auf Download

3. SPIEL STARTEN
   • Klicke "Mit Profil spielen"
   • Steam startet automatisch

VORAUSSETZUNGEN:
• Steam installiert
• r2modman empfohlen
• Internetverbindung

PROBLEME:
• Admin-Modus für Konfiguration
• Profile in r2modman-Ordner
• Steam-Overlay für beste Erfahrung

HINWEIS:
Profile werden automatisch in die
richtige Verzeichnisstruktur installiert.`,
		Close: "Schließen",

		Error:            "Fehler",
		SteamNotFound:    "Steam nicht gefunden",
		R2ModmanNotFound: "r2modman nicht gefunden",
		NetworkError:     "Netzwerkfehler",
	}
}
