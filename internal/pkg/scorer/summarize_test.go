//nolint:testpackage // exercises the unexported splitSentences helper
package scorer

import "testing"

func TestSplitSentences(t *testing.T) {
	t.Parallel()

	body := "Too short to keep. " +
		"This sentence is comfortably longer than the minimum length threshold here! " +
		"And here is another sentence that also exceeds the minimum length requirement."

	got := splitSentences(body)

	if len(got) != 2 {
		t.Fatalf("got %d sentences, want 2: %#v", len(got), got)
	}

	for _, sentence := range got {
		if len(sentence) < minSentenceLen {
			t.Errorf("kept a too-short sentence: %q", sentence)
		}
	}
}
