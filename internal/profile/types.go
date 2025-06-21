package profile

import "sync"

var (
	packageCache = make(map[string]*ThunderstorePackage)
	cacheMutex   sync.RWMutex
)

type ThunderstorePackage struct {
	FullName     string                       `json:"full_name"`
	Name         string                       `json:"name"`
	Owner        string                       `json:"owner"`
	Versions     []ThunderstorePackageVersion `json:"versions"`
	PackageURL   string                       `json:"package_url"`
	IsDeprecated bool                         `json:"is_deprecated"`
	Latest       *ThunderstorePackageVersion  `json:"-"`
}

type ThunderstorePackageVersion struct {
	VersionNumber string   `json:"version_number"`
	DownloadURL   string   `json:"download_url"`
	Dependencies  []string `json:"dependencies"`
	FileSize      int64    `json:"file_size"`
	FullName      string   `json:"full_name"`
	Description   string   `json:"description"`
}

type ThunderstoreManifest struct {
	Name          string   `json:"name"`
	VersionNumber string   `json:"version_number"`
	WebsiteURL    string   `json:"website_url"`
	Description   string   `json:"description"`
	Dependencies  []string `json:"dependencies"`
}

type ModDependency struct {
	FullName string
	Version  string
	Required bool
}

type ExportR2X struct {
	ProfileName string   `yaml:"profileName"`
	Mods        []R2XMod `yaml:"mods"`
}

type R2XMod struct {
	Name    string     `yaml:"name"`
	Version R2XVersion `yaml:"version"`
	Enabled bool       `yaml:"enabled"`
}

type R2XVersion struct {
	Major int `yaml:"major"`
	Minor int `yaml:"minor"`
	Patch int `yaml:"patch"`
}

type ModEntry struct {
	ManifestVersion      int           `yaml:"manifestVersion"`
	Name                 string        `yaml:"name"`
	AuthorName           string        `yaml:"authorName"`
	WebsiteURL           string        `yaml:"websiteUrl"`
	DisplayName          string        `yaml:"displayName"`
	Description          string        `yaml:"description"`
	GameVersion          string        `yaml:"gameVersion"`
	NetworkMode          string        `yaml:"networkMode"`
	PackageType          string        `yaml:"packageType"`
	InstallMode          string        `yaml:"installMode"`
	InstalledAtTime      int64         `yaml:"installedAtTime"`
	Loaders              []string      `yaml:"loaders"`
	Dependencies         []string      `yaml:"dependencies"`
	Incompatibilities    []string      `yaml:"incompatibilities"`
	OptionalDependencies []string      `yaml:"optionalDependencies"`
	VersionNumber        VersionNumber `yaml:"versionNumber"`
	Enabled              bool          `yaml:"enabled"`
	Icon                 string        `yaml:"icon"`
}

type VersionNumber struct {
	Major int `yaml:"major"`
	Minor int `yaml:"minor"`
	Patch int `yaml:"patch"`
}

type ModsYML []ModEntry

type ExportFormatR2X struct {
	ProfileName string    `yaml:"profileName"`
	Mods        []ModInfo `yaml:"mods"`
}

type ModInfo struct {
	Name    string `yaml:"name"`
	Version struct {
		Major int `yaml:"major"`
		Minor int `yaml:"minor"`
		Patch int `yaml:"patch"`
	} `yaml:"version"`
	Enabled bool `yaml:"enabled"`
}

type ModReference struct {
	PackageName struct {
		Namespace string `yaml:"namespace"`
		Name      string `yaml:"name"`
	} `yaml:"packageName"`
	Version string `yaml:"version"`
	Enabled bool   `yaml:"enabled"`
}

type ProfileVersion struct {
	URL     string `json:"url"`
	Version string `json:"version"`
}

type ProfileStatus struct {
	Installed    bool
	UpToDate     bool
	HasUpdate    bool
	InstallError error
	VersionError error
}
