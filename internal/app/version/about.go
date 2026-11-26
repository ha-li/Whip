package version

// About is bound to the frontend and reports information about the app.
type About struct {
	version Version
}

// NewAbout returns an About for the given app version.
func NewAbout(v Version) *About {
	return &About{version: v}
}

// Version returns the full app version, e.g. "0.1.2-alpha".
func (a *About) Version() string {
	return a.version.String()
}
