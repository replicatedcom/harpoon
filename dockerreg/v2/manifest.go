package v2

// Manifest represents a remote v2 image manifest file
type Manifest struct {
	FSLayers []fslayer `json:"fsLayers"`
	History  []history `json:"history"`
}

type fslayer struct {
	BlobSum string `json:"blobSum"`
}

type history struct {
	Compatibility string `json:"v1Compatibility"`
}
