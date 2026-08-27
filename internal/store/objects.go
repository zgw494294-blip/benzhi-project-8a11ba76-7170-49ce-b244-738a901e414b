package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/benzhi/oral-history-release/internal/domain"
)

type evidenceObject struct {
	Kind    string
	Payload any
}

func evidenceObjects(c *domain.OralHistoryCase) []evidenceObject {
	objects := make([]evidenceObject, 0, len(c.Consents)+len(c.Transcripts)+len(c.Findings)+len(c.Objections)+len(c.Credentials)+1)
	for _, value := range c.Consents {
		objects = append(objects, evidenceObject{Kind: "consent", Payload: value})
	}
	for _, value := range c.Transcripts {
		objects = append(objects, evidenceObject{Kind: "transcript", Payload: value})
	}
	for _, value := range c.Findings {
		objects = append(objects, evidenceObject{Kind: "finding", Payload: value})
	}
	for _, value := range c.Objections {
		objects = append(objects, evidenceObject{Kind: "objection", Payload: value})
	}
	if c.FrozenManifest != nil {
		objects = append(objects, evidenceObject{Kind: "manifest", Payload: *c.FrozenManifest})
	}
	for _, value := range c.Credentials {
		objects = append(objects, evidenceObject{Kind: "credential", Payload: value})
	}
	return objects
}

func (r *Repository) persistObjects(c *domain.OralHistoryCase) ([]string, error) {
	objects := evidenceObjects(c)
	digests := make([]string, 0, len(objects))
	for _, object := range objects {
		payload, err := json.Marshal(object.Payload)
		if err != nil {
			return nil, err
		}
		digest := domain.DigestBytes(payload)
		digests = append(digests, digest)
		path := filepath.Join(r.root, "objects", digest+".json")
		r.objectMu.Lock()
		_, known := r.knownObject[path]
		r.objectMu.Unlock()
		if known {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			r.objectMu.Lock()
			r.knownObject[path] = struct{}{}
			r.objectMu.Unlock()
			continue
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		envelope := objectEnvelope{SchemaVersion: schemaVersion, Kind: object.Kind, Digest: digest, Payload: payload}
		data, err := json.MarshalIndent(envelope, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := atomicWrite(path, data, 0o640); err != nil {
			return nil, err
		}
		r.objectMu.Lock()
		r.knownObject[path] = struct{}{}
		r.objectMu.Unlock()
	}
	return digests, nil
}

func (r *Repository) validateObject(digest string) error {
	data, err := os.ReadFile(filepath.Join(r.root, "objects", digest+".json"))
	if err != nil {
		return fmt.Errorf("读取证据对象 %s: %w", digest, err)
	}
	var envelope objectEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("解析证据对象 %s: %w", digest, err)
	}
	if envelope.SchemaVersion != schemaVersion || envelope.Digest != digest || domain.DigestBytes(envelope.Payload) != digest {
		return fmt.Errorf("证据对象 %s 摘要或 schemaVersion 不匹配", digest)
	}
	return nil
}
