package importer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/replicatedcom/harpoon/log"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/image"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type v1Store struct {
	Workspace string
}

// Copied from docker
type manifestItem struct {
	Config   string
	RepoTags []string
	Layers   []string
	Parent   image.ID `json:",omitempty"`
}

func getV1Store(verifiedManifest *schema1.Manifest) (*v1Store, error) {
	result := &v1Store{}

	compat := &v1Compatibility{}
	// first entry is the top layer
	if err := json.Unmarshal([]byte(verifiedManifest.History[0].V1Compatibility), compat); err != nil {
		log.Error(err)
		return nil, err
	}

	if compat.ID == "" {
		err := errors.New("Compatibility info has no image ID")
		log.Error(err)
		return nil, err
	}

	dir, err := ioutil.TempDir("", "harpoon")
	if err != nil {
		return nil, err
	}
	result.Workspace = dir

	return result, nil
}

func (repo *v1Store) delete() error {
	if err := os.RemoveAll(repo.Workspace); err != nil {
		return errors.Wrap(err, "failed to remove workspace")
	}
	return nil
}

func (repo *v1Store) writeConfigFile(imageID image.ID, config []byte) error {
	filename := filepath.Join(repo.Workspace, fmt.Sprintf("%s.json", digest.Digest(imageID).Hex()))
	if err := ioutil.WriteFile(filename, config, 0644); err != nil {
		return errors.Wrap(err, "failed to write config file")
	}

	return nil
}

func (repo *v1Store) writeRepositoriesFile(ref reference.Named, imageID image.ID) error {
	filename := filepath.Join(repo.Workspace, "repositories")

	tagged, ok := ref.(reference.NamedTagged)
	if !ok {
		return errors.Errorf("reference is not tagged: %T", ref)
	}

	repos := map[string]interface{}{
		ref.Name(): map[string]string{
			tagged.Tag(): digest.Digest(imageID).Hex(),
		},
	}

	contents, err := json.Marshal(repos)
	if err != nil {
		return errors.Wrap(err, "failed to marshal repositories")
	}

	if err := ioutil.WriteFile(filename, contents, 0644); err != nil {
		return errors.Wrap(err, "failed to write repositories file")
	}

	return nil
}

func (repo *v1Store) writeManifestFile(ref reference.Named, imageID image.ID, layerV1IDs []digest.Digest) error {
	filename := filepath.Join(repo.Workspace, "manifest.json")

	layers := make([]string, 0)
	for _, V1ID := range layerV1IDs {
		layers = append(layers, filepath.Join(digest.Digest(V1ID).Hex(), "layer.tar"))
	}

	manifest := manifestItem{
		Config:   digest.Digest(imageID).Hex() + ".json",
		RepoTags: []string{ref.String()},
		Layers:   layers,
		// TODO: ParentID is probbaly empty, but when is it not empty?
	}

	contents, err := json.Marshal([]manifestItem{manifest})
	if err != nil {
		return errors.Wrap(err, "failed to marshal manifest")
	}

	if err := ioutil.WriteFile(filename, contents, 0644); err != nil {
		return errors.Wrap(err, "failed to write manifest file")
	}

	return nil
}

// Copied from docker
func verifySchema1Manifest(signedManifest *schema1.SignedManifest, ref reference.Named) (m *schema1.Manifest, err error) {
	if digested, isCanonical := ref.(reference.Canonical); isCanonical {
		verifier := digested.Digest().Verifier()
		if _, err := verifier.Write(signedManifest.Canonical); err != nil {
			return nil, errors.Wrap(err, "failed to write verifier")
		}
		if !verifier.Verified() {
			return nil, errors.Errorf("image verification failed for digest %s", digested.Digest())
		}
	}
	m = &signedManifest.Manifest

	if m.SchemaVersion != 1 {
		return nil, errors.Errorf("unsupported schema version %d for %q", m.SchemaVersion, ref.String())
	}
	if len(m.FSLayers) != len(m.History) {
		return nil, errors.Errorf("length of history not equal to number of layers for %q", ref.String())
	}
	if len(m.FSLayers) == 0 {
		return nil, errors.Errorf("no FSLayers in manifest for %q", ref.String())
	}

	return m, nil
}
