package x

import "testing"

func TestParseArticleRefAcceptsURLAndBareID(t *testing.T) {
	for _, ref := range []string{"2103118352554344448", "https://x.com/i/article/2103118352554344448", "x://article/2103118352554344448"} {
		id, err := ParseArticleRef(ref)
		if err != nil || id != "2103118352554344448" {
			t.Errorf("ParseArticleRef(%q) = %q, %v", ref, id, err)
		}
	}
	if _, err := ParseArticleRef("https://x.com/leo/status/2103529310208606701"); err == nil {
		t.Error("a tweet URL was accepted as an article")
	}
}

func TestArticleFieldsAreFirstClass(t *testing.T) {
	if !containsString(FieldKinds, KindArticle) {
		t.Fatalf("FieldKinds = %v, want article", FieldKinds)
	}
	got := map[string]bool{}
	for _, f := range Fields(KindArticle) {
		got[f.Name] = true
		if tier, ok := f.Tier(); !ok || tier != 2 {
			t.Errorf("article field %s tier = %d, %v; want session tier", f.Name, tier, ok)
		}
	}
	for _, want := range []string{"title", "body", "content_state", "cover", "media", "author", "linked_post"} {
		if !got[want] {
			t.Errorf("article fields missing %q: %v", want, got)
		}
	}
}

func TestArticleGraphCarriesArticleAuthorAndLinkedPost(t *testing.T) {
	a := &Article{Title: "Article", Author: NewUser("leo"), LinkedPost: NewTweet("20")}
	a.Identify(KindArticle, "10")
	a.Stamp(7, "https://x.com/i/api/graphql/TweetDetail")
	doc := Graph(a)
	if len(doc.Nodes) != 3 {
		t.Fatalf("nodes = %+v, want article, author, and post", doc.Nodes)
	}
	want := map[string]bool{
		"x://user/leo authored x://article/10": false,
		"x://tweet/20 links_to x://article/10": false,
	}
	for _, edge := range doc.Edges {
		key := edge.From + " " + string(edge.Predicate) + " " + edge.To
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for edge, found := range want {
		if !found {
			t.Errorf("missing edge %q in %+v", edge, doc.Edges)
		}
	}
	var articleRecord bool
	for _, node := range doc.Nodes {
		if node.URI == "x://article/10" && node.Record == a {
			articleRecord = true
		}
	}
	if !articleRecord {
		t.Error("article record was not carried into the graph")
	}
}

func containsString(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
