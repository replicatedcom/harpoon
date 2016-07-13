package dockerreg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/replicatedcom/harpoon/log"

	"github.com/docker/distribution/digest"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/image"
	"github.com/docker/docker/reference"
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

func getV1Store(manifest *schema1.SignedManifest, dockerRemote *DockerRemote) (*v1Store, error) {
	result := &v1Store{}

	verifiedManifest, err := verifySchema1Manifest(manifest, dockerRemote.Ref)
	if err != nil {
		return nil, err
	}

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
		log.Error(err)
		return err
	}
	return nil
}

func (repo *v1Store) writeConfigFile(dockerRemote *DockerRemote, imageID image.ID, config []byte) error {
	filename := filepath.Join(repo.Workspace, fmt.Sprintf("%s.json", digest.Digest(imageID).Hex()))
	if err := ioutil.WriteFile(filename, config, 0644); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (repo *v1Store) writeRepositoriesFile(dockerRemote *DockerRemote, imageID image.ID) error {
	filename := filepath.Join(repo.Workspace, "repositories")

	tagged, ok := dockerRemote.Ref.(reference.NamedTagged)
	if !ok {
		err := fmt.Errorf("Reference is not tagged: %T", dockerRemote.Ref)
		log.Error(err)
		return err
	}

	repos := map[string]interface{}{
		dockerRemote.Ref.Name(): map[string]string{
			tagged.Tag(): digest.Digest(imageID).Hex(),
		},
	}

	contents, err := json.Marshal(repos)
	if err != nil {
		log.Error(err)
		return err
	}

	if err := ioutil.WriteFile(filename, contents, 0644); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (repo *v1Store) writeManifestFile(dockerRemote *DockerRemote, imageID image.ID, layerV1IDs []digest.Digest) error {
	filename := filepath.Join(repo.Workspace, "manifest.json")

	layers := make([]string, 0)
	for _, V1ID := range layerV1IDs {
		layers = append(layers, filepath.Join(digest.Digest(V1ID).Hex(), "layer.tar"))
	}

	manifest := manifestItem{
		Config:   digest.Digest(imageID).Hex() + ".json",
		RepoTags: []string{dockerRemote.Ref.String()},
		Layers:   layers,
		// TODO: ParentID is probbaly empty, but when is it not empty?
	}

	contents, err := json.Marshal([]manifestItem{manifest})
	if err != nil {
		log.Error(err)
		return err
	}

	if err := ioutil.WriteFile(filename, contents, 0644); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// Copied from docker
func verifySchema1Manifest(signedManifest *schema1.SignedManifest, ref reference.Named) (m *schema1.Manifest, err error) {
	if digested, isCanonical := ref.(reference.Canonical); isCanonical {
		verifier, err := digest.NewDigestVerifier(digested.Digest())
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if _, err := verifier.Write(signedManifest.Canonical); err != nil {
			log.Error(err)
			return nil, err
		}
		if !verifier.Verified() {
			err := fmt.Errorf("image verification failed for digest %s", digested.Digest())
			log.Error(err)
			return nil, err
		}
	}
	m = &signedManifest.Manifest

	if m.SchemaVersion != 1 {
		err := fmt.Errorf("unsupported schema version %d for %q", m.SchemaVersion, ref.String())
		log.Error(err)
		return nil, err
	}
	if len(m.FSLayers) != len(m.History) {
		err := fmt.Errorf("length of history not equal to number of layers for %q", ref.String())
		log.Error(err)
		return nil, err
	}
	if len(m.FSLayers) == 0 {
		err := fmt.Errorf("no FSLayers in manifest for %q", ref.String())
		log.Error(err)
		return nil, err
	}

	return m, nil
}
