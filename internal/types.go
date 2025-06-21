package internal

const (
	WindowWidth  = 600
	WindowHeight = 400
	AppName      = "Wesleys Profiles"
	AppVersion   = "v1.0.16"
	AppID        = "fyi.wesley.modhelper"
)

type Config struct {
	ManifestURL string `json:"manifest_url"`
	TargetDir   string `json:"target_dir"`
}

type Game struct {
	Name            string   `json:"name"`
	ID              string   `json:"id"`
	Header          string   `json:"icon"`
	ProfileName     string   `json:"profileName"`
	URL             string   `json:"url"`
	LaunchArgs      string   `json:"launchArgs"`
	Community       string   `json:"community"`
	ExecutableNames []string `json:"executableNames"`
	Version         string   `json:"version"`
}
