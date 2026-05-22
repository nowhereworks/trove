package packages

func orderManifestFirst(artifacts []UploadArchiveArtifact) []UploadArchiveArtifact {
	ordered := make([]UploadArchiveArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Path == "trove.yaml" {
			ordered = append(ordered, artifact)
		}
	}
	for _, artifact := range artifacts {
		if artifact.Path != "trove.yaml" {
			ordered = append(ordered, artifact)
		}
	}
	return ordered
}
