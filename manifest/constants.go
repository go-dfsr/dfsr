package manifest

const (
	// Dir is the directory that typically contains the conflict and deleted
	// manifest.
	Dir = `DfsrPrivate`
	// File is the name of the typical conflict and deleted manifest file.
	File = "ConflictAndDeletedManifest.xml"
	// Path is the last part of a typical conflict and deleted manifset file path.
	Path = `\` + Dir + `\` + File
)
