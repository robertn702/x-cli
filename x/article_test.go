package x

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// This is the smallest sanitized TweetDetail shape that carries an X Article.
// It is intentionally synthetic: committed fixtures must never contain session
// cookies or an unfiltered response captured with a user's account.
const articleTweetDetail = `{
  "data": {
    "threaded_conversation_with_injections_v2": {
      "instructions": [{
        "entries": [{
          "content": {
            "itemContent": {
              "tweet_results": {
                "result": {
                  "__typename": "Tweet",
                  "rest_id": "2103529310208606701",
                  "legacy": {
                    "id_str": "2103529310208606701",
                    "full_text": "A tweet linking to an article"
                  },
                  "article": {
                    "article_results": {
                      "result": {
                        "rest_id": "2103118352554344448",
                        "title": "A useful article",
                        "plain_text": "First paragraph.\n\nSecond paragraph.",
                        "content_state": {"blocks": [{"text": "First paragraph."}], "entityMap": [{"key": "0", "value": {"type": "LINK", "data": {"url": "https://example.com"}}}]},
                        "cover_media": {"media_key": "3_1000", "media_info": {"__typename": "ApiImage", "original_img_url": "https://pbs.twimg.com/media/cover.jpg"}},
                        "media_entities": [
                          {
                            "media_id": "1001",
                            "media_key": "3_1001",
                            "media_info": {
                              "__typename": "ApiImage",
                              "original_img_url": "https://pbs.twimg.com/media/example.jpg",
                              "original_img_width": 1200,
                              "original_img_height": 675
                            }
                          },
                          {
                            "media_id": "1002",
                            "media_key": "13_1002",
                            "media_info": {
                              "__typename": "ApiVideo",
                              "duration_millis": 4200,
                              "preview_image": {
                                "original_img_url": "https://pbs.twimg.com/media/preview.jpg",
                                "original_img_width": 1280,
                                "original_img_height": 720
                              },
                              "variants": [{
                                "bit_rate": 832000,
                                "content_type": "video/mp4",
                                "url": "https://video.twimg.com/article/example.mp4"
                              }]
                            }
                          },
                          {
                            "media_key": "13_1003",
                            "media_info": {"__typename": "ApiGif", "preview_image": {"original_img_url": "https://pbs.twimg.com/media/gif.jpg"}, "variants": [{"content_type": "video/mp4", "url": "https://video.twimg.com/article/gif.mp4"}]}
                          }
                        ]
                      }
                    }
                  }
                }
              }
            }
          }
        }]
      }]
    }
  }
}`

func TestTweetDetailExposesLinkedArticle(t *testing.T) {
	tweets, _ := collectTweets([]byte(articleTweetDetail))
	if len(tweets) != 1 {
		t.Fatalf("got %d tweets, want 1", len(tweets))
	}
	article := tweets[0].Article
	if article == nil {
		t.Fatal("linked article was discarded")
	}
	if article.ID != "2103118352554344448" {
		t.Errorf("article id = %q", article.ID)
	}
	if article.Title != "A useful article" {
		t.Errorf("article title = %q", article.Title)
	}
	if article.Body != "First paragraph.\n\nSecond paragraph." {
		t.Errorf("article body = %q", article.Body)
	}
	if article.URL != "https://x.com/i/article/2103118352554344448" {
		t.Errorf("article url = %q", article.URL)
	}
	if len(article.ContentState) == 0 || !strings.Contains(string(article.ContentState), "https://example.com") {
		t.Errorf("article structured content = %s", article.ContentState)
	}
	if article.Cover == nil || article.Cover.URL != "https://pbs.twimg.com/media/cover.jpg" {
		t.Errorf("article cover = %+v", article.Cover)
	}
	if len(article.Media) != 3 {
		t.Fatalf("got %d article media, want 3", len(article.Media))
	}
	image, video := article.Media[0], article.Media[1]
	if image.Type != "photo" || image.URL != "https://pbs.twimg.com/media/example.jpg" || image.Width != 1200 || image.Height != 675 {
		t.Errorf("image = %+v", image)
	}
	if video.Type != "video" || video.Preview != "https://pbs.twimg.com/media/preview.jpg" || video.Duration != 4200 {
		t.Errorf("video = %+v", video)
	}
	if len(video.Variants) != 1 || video.Variants[0].Bitrate != 832000 || video.Variants[0].URL != "https://video.twimg.com/article/example.mp4" {
		t.Errorf("video variants = %+v", video.Variants)
	}
	if gif := article.Media[2]; gif.Type != "animated_gif" || len(gif.Variants) != 1 {
		t.Errorf("gif = %+v", gif)
	}

	encoded, err := json.Marshal(tweets[0])
	if err != nil {
		t.Fatal(err)
	}
	var visible map[string]any
	if err := json.Unmarshal(encoded, &visible); err != nil {
		t.Fatal(err)
	}
	if _, ok := visible["article"]; !ok {
		t.Fatal("article is absent from the tweet JSON returned by the CLI")
	}
}

type articleTransport struct {
	operations []string
}

func (f *articleTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Path == "/" {
		return reply(r, "<html></html>", "text/html", "")
	}
	if strings.Contains(r.URL.Path, "/i/api/graphql/") {
		f.operations = append(f.operations, r.URL.Path)
	}
	if strings.HasSuffix(r.URL.Path, "/TweetDetail") {
		return reply(r, articleTweetDetail, "application/json", "")
	}
	return reply(r, "", "application/json", "")
}

func TestSessionTierTweetUsesTweetDetailForArticleBody(t *testing.T) {
	cfg := fixtureCfg(t)
	cfg.AuthToken = "test-auth-token"
	cfg.CT0 = "test-csrf-token"
	cfg.Tier = "session"
	c := NewClient(cfg)
	transport := &articleTransport{}
	c.hc.Transport = transport
	g := NewGraphQL(c, NewSession(cfg), cfg)

	tweet, err := g.TweetByID(context.Background(), "2103529310208606701")
	if err != nil {
		t.Fatalf("TweetByID: %v", err)
	}
	if tweet.Article == nil || tweet.Article.Body == "" {
		t.Fatalf("signed-in tweet has no article body: %+v", tweet.Article)
	}
	if len(transport.operations) != 1 || !strings.HasSuffix(transport.operations[0], "/TweetDetail") {
		t.Errorf("GraphQL operations = %v, want TweetDetail", transport.operations)
	}
}

func TestGuestTierHonoredWithStoredSession(t *testing.T) {
	cfg := fixtureCfg(t)
	cfg.AuthToken = "test-auth-token"
	cfg.CT0 = "test-csrf-token"
	cfg.Tier = "guest"
	c := NewClient(cfg)
	transport := &articleTransport{}
	c.hc.Transport = transport
	g := NewGraphQL(c, NewSession(cfg), cfg)
	_, _ = g.TweetByID(context.Background(), "2103529310208606701")
	if len(transport.operations) == 0 || !strings.HasSuffix(transport.operations[0], "/TweetResultByRestId") {
		t.Errorf("guest tier operations = %v, want TweetResultByRestId", transport.operations)
	}
}

func TestThreadIncludesLinkedArticle(t *testing.T) {
	cfg := fixtureCfg(t)
	cfg.AuthToken = "test-auth-token"
	cfg.CT0 = "test-csrf-token"
	c := NewClient(cfg)
	c.hc.Transport = &articleTransport{}
	g := NewGraphQL(c, NewSession(cfg), cfg)

	var got *Tweet
	err := g.Thread(context.Background(), "2103529310208606701", 1, func(tweet *Tweet) error {
		got = tweet
		return nil
	})
	if err != nil {
		t.Fatalf("Thread: %v", err)
	}
	if got == nil || got.Article == nil || got.Article.Title != "A useful article" {
		t.Fatalf("thread tweet article = %+v", got)
	}
}
