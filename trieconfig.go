package trieconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode"
)

type (
	// Section represents the generic nested config container. SectionType
	// can be either 'COLLECTION' or 'ITEM'.
	Section struct {
		Name        string          `json:"name,omitempty"`
		Children    []*Section      `json:"section"`
		Src         json.RawMessage `json:"src"`
		SectionType SectionType     `json:"type,omitempty"`
		ResourceID  string          `json:"resourceId,omitempty"`
		requestPath string
	}

	// SectionType enum
	SectionType int

	//ConfigGetter - Hosts a Section Trie, populated from config src, implements http.Handler
	ConfigGetter struct {
		src           io.Reader
		configSection *Section
	}
)

// SectionType Enum
const (
	COLLECTION SectionType = iota
	ITEM
	WCMS_TOPIC
	RECCOMENDATIONS_TOPIC
	TOPIC_CONTENT
)

// UnmarshalJSON - SectionType enum unmarshalling
func (s *SectionType) UnmarshalJSON(b []byte) error {
	ctype := strings.ToLower(string(b[1 : len(b)-1]))
	switch ctype {
	case "recommendations":
		*s = RECCOMENDATIONS_TOPIC
		return nil
	case "wcmstopic":
		*s = WCMS_TOPIC
		return nil
	case "item":
		*s = ITEM
		return nil
	case "topiccontent":
		*s = TOPIC_CONTENT
		return nil
	case "section":
		*s = COLLECTION
		return nil
	default:
		return errors.New("Could not Parse section type ")
	}
}

// ResourceLocator returns Section's resource Locator
func (s *Section) ResourceLocator() string {
	return s.ResourceID
}

// Traverse recursively searches the Section Trie for Sections By ResourceId
func (s *Section) Traverse(pathSegments []string) *Section {
	pathSegment := pathSegments[0]

	if len(s.Children) > 0 {
		for _, child := range s.Children {
			if pathSegment == child.ResourceLocator() { //we have a match
				nextSegments := pathSegments[1:]
				if len(nextSegments) > 0 { // if there are more segments in the path
					return child.Traverse(nextSegments) //recurse
				}
				return child
			}
		}
	}
	return s
}

func slugify(r rune) rune {
	if unicode.IsSpace(r) {
		return '_'
	}
	return r
}

// UnmarshalJSON - control how Sections are serialised from JSON
func (s *Section) UnmarshalJSON(b []byte) error {
	type Alias Section
	var alias Alias
	if err := json.Unmarshal(b, &alias); err != nil {
		return err
	}

	if alias.ResourceID == "" {
		alias.ResourceID = alias.Name
	}
	alias.ResourceID = strings.Map(slugify, strings.ToLower(alias.ResourceID))
	*s = Section(alias)
	s.Src = json.RawMessage(b)
	return nil
}

// UpdateResourceRoutes recursively updates Sections' requestPaths - by appending
// the parent's requestPath in front of the child's Resource Identifier
func (s *Section) updateResourceRoutes() {
	for _, section := range s.Children {
		section.requestPath = path.Join(s.requestPath, section.ResourceID)
		section.updateResourceRoutes()
	}
}

func (s *Section) Map(obj interface{}) error {
	return json.Unmarshal(s.Src, obj)
}

// NewConfigGetter returns a ConfigGetter, given an io.Reader that returns a valid section config
func NewConfigGetter(configSource io.Reader) (*ConfigGetter, error) {
	c := &ConfigGetter{
		configSection: &Section{},
	}
	err := c.readConfig(configSource)
	return c, err
}

// Get searches Section Trie by request URI and returns the corresponding HalResource
func (cr *ConfigGetter) Get(requestURI string) (*Section, error) {
	pathSegments := strings.Split(requestURI, "/")[1:]

	if len(pathSegments) < 1 {
		return nil, fmt.Errorf("request URI: %s, doesn't contain enough pathSegments", requestURI)
	}
	endSegment := pathSegments[len(pathSegments)-1]

	foundSection := cr.configSection.Traverse(pathSegments)
	if foundSection.ResourceLocator() != endSegment {
		return nil, fmt.Errorf("Found resource.ResourceId:%s, does not match request URI endSegment:%s", foundSection.ResourceLocator(), endSegment)
	}

	return foundSection, nil
}

// ReadConfig populates ConfigReaders Section Trie from configSource
func (cr *ConfigGetter) readConfig(configSource io.Reader) error {
	cr.src = configSource
	dec := json.NewDecoder(configSource)

	err := dec.Decode(cr.configSection)
	cr.configSection.updateResourceRoutes()
	return err
}
