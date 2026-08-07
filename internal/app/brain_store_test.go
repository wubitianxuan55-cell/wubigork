package app

import (
	"strings"
	"testing"
)

func TestBrainStoreReadWriteSearch(t *testing.T) {
	mainFake := &fakeMainBrain{rows: map[string]string{}}
	rightFake := &fakeRightBrain{rows: map[string]string{}}
	leftFake := &fakeLeftBrain{rows: map[string]string{}}
	bs := &BrainStore{main: mainFake, left: leftFake, right: rightFake}

	if err := bs.Write("brain.right", "甲方A", "偏好", "保守报价"); err != nil {
		t.Fatal(err)
	}
	hits, err := bs.Search("甲方A 报价", "brain.right")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || !strings.Contains(hits[0].Text, "保守报价") {
		t.Fatalf("hits = %+v", hits)
	}
}

func TestBrainStoreLinkAndCrossRefs(t *testing.T) {
	mainFake := &fakeMainBrain{rows: map[string]string{}}
	bs := &BrainStore{main: mainFake, left: &fakeLeftBrain{}, right: &fakeRightBrain{}, links: NewLinkStore(nil)}
	if err := bs.Link("甲方A", "brain.right", "fact-1"); err != nil {
		t.Fatal(err)
	}
	if err := bs.Link("甲方A", "brain.left", "proposal:p1"); err != nil {
		t.Fatal(err)
	}
	refs, err := bs.CrossRefs("甲方A")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("refs = %+v", refs)
	}
}

type fakeMainBrain struct{ rows map[string]string }

func (f *fakeMainBrain) Read(entity string) ([]Fact, error) { return nil, nil }
func (f *fakeMainBrain) Write(entity, attribute, value string) error {
	f.rows[entity+"|"+attribute] = value
	return nil
}
func (f *fakeMainBrain) Search(query string) ([]Hit, error) { return nil, nil }

type fakeRightBrain struct{ rows map[string]string }

func (f *fakeRightBrain) Read(entity string) ([]Fact, error) { return nil, nil }
func (f *fakeRightBrain) Write(entity, attribute, value string) error {
	f.rows[entity+"|"+attribute] = value
	return nil
}
func (f *fakeRightBrain) Search(query string) ([]Hit, error) {
	terms := strings.Fields(query)
	for k, v := range f.rows {
		for _, term := range terms {
			if strings.Contains(k, term) || strings.Contains(v, term) {
				return []Hit{{Brain: "brain.right", Entity: strings.SplitN(k, "|", 2)[0], Text: v}}, nil
			}
		}
	}
	return nil, nil
}

type fakeLeftBrain struct{ rows map[string]string }

func (f *fakeLeftBrain) Read(entity string) ([]Fact, error) { return nil, nil }
func (f *fakeLeftBrain) Write(entity, attribute, value string) error { return nil }
func (f *fakeLeftBrain) Search(query string) ([]Hit, error) { return nil, nil }
