package manifest

const (
	// StandardDir is the directory that typically contains the conflict and
	// deleted manifest.
	StandardDir = "DfsrPrivate"
	// StandardFile is the name of the typical conflict and deleted manifest file.
	StandardFile = "ConflictAndDeletedManifest.xml"
	// StandardPath is the last part of a typical conflict and deleted manifest
	// file path.
	StandardPath = `\` + StandardDir + `\` + StandardFile
)
