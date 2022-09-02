package main

import (
	"encoding/json"
	"html"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml"
)

type issueLabel struct {
	Name string `json:"name"`
}

type issueEvent struct {
	Action string `json:"action"`
	Issue  struct {
		Title  string       `json:"title"`
		Number int          `json:"number"`
		Body   string       `json:"body"`
		Labels []issueLabel `json:"labels"`
	} `json:"issue"`
}

func isGeneratorIssue(labels []issueLabel) bool {
	for _, l := range labels {
		if l.Name == "generator" {
			return true
		}
	}
	return false
}

type genConfig struct {
	Name      string `toml:"name" json:"-"`
	IssueID   string `toml:"id" json:"-"`
	Variables []struct {
		ID          string `toml:"id" json:"id"`
		Description string `toml:"description" json:"description"`
		Default     string `toml:"default" json:"default"`
	} `toml:"variables" json:"variables"`
	Templates map[string][]string `toml:"templates" json:"templates"`
}

func main() {
	eventFile := os.Getenv("GITHUB_EVENT_PATH")
	if eventFile == "" {
		log.Fatal("no event file")
	}
	f, err := ioutil.ReadFile(eventFile)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("event:", string(f))

	var event issueEvent
	err = json.Unmarshal(f, &event)
	if err != nil {
		log.Fatal(err)
	}

	switch event.Action {
	case "opened", "edited", "reopened", "labeled":
		begin := strings.Index(event.Issue.Body, "```toml")
		if begin == -1 {
			log.Fatal("cannot find toml code block in issue body")
		}
		begin += 7
		end := strings.Index(event.Issue.Body[begin:], "```")
		if end == -1 {
			log.Fatal("cannot find toml code block in issue body")
		}
		if !isGeneratorIssue(event.Issue.Labels) {
			log.Print("not generator issue, skip")
			return
		}
		var config genConfig
		if err = toml.Unmarshal([]byte(event.Issue.Body[begin:begin+end]), &config); err != nil {
			log.Fatal("invalid toml", err)
		}
		config.Name = event.Issue.Title
		config.IssueID = strconv.Itoa(event.Issue.Number)
		if err := genPage(&config); err != nil {
			log.Fatal(err)
		}
	case "deleted", "closed", "unlabeled":
		if event.Action == "unlabeled" && isGeneratorIssue(event.Issue.Labels) {
			log.Print("is still generator issue, skip")
			return
		}
		err = os.Remove(filepath.Join("docs", strconv.Itoa(event.Issue.Number)+".html"))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func genPage(config *genConfig) error {
	j, err := json.Marshal(config)
	if err != nil {
		return err
	}
	s := strings.ReplaceAll(pageTpl, "{{.Title}}", html.EscapeString(config.Name))
	s = strings.ReplaceAll(s, "{{.JSON}}", string(j))
	return ioutil.WriteFile(filepath.Join("docs", config.IssueID+".html"), []byte(s), 0644)
}

var pageTpl = `<!DOCTYPE HTML>
<html lang="zh-CN" style="height: 100%">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <link href="https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/4.5.3/css/bootstrap.min.css" rel="stylesheet">
    <title>{{.Title}} | 骚话生成器生成器</title>
  </head>
  <body style="height: 100%">
    <div class="container" id="app" style="min-height: 100%;">
      <nav class="navbar navbar-expand-lg navbar-light bg-light">
        <a class="navbar-brand" href="#">{{.Title}}</a>
        <div class="ml-auto">
        <a href="https://github.com/disksing/sao-gen-gen/issues/new?labels=generator&template=generator.md" target="_blank">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-github" viewBox="0 0 16 16">
              <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.012 8.012 0 0 0 16 8c0-4.42-3.58-8-8-8z"/>
            </svg>
            创建新的生成器
          </a>
        </div>
      </nav>
      <div class="row mx-0 mt-3" style="padding: 20px; padding-bottom: 100px;">
        <form class="col-12">
          <div class="form-group row" v-for="v in variables">
            <label class="col-sm-3 col-form-label">{{ v.description }}</label>
            <div class="col-sm-9">
              <input type="text" class="form-control" :placeholder="v.default" :key="v.id" v-model="v.value" />
            </div>
          </div>
        </form>
        <div class="col-12"><hr/></div>
        <div class="row col-12 mx-0 mt-3">
          <div class="ml-auto">
            <button type="button" class="btn btn-outline-primary mx-0 mr-0" v-on:click="updateTpl">重新生成</button>
            <button type="button" class="btn btn-outline-primary mx-0 mr-0" v-on:click="copy">复制至剪贴板</button>
            <button type="button" class="btn btn-outline-primary mx-0 mr-0" @click.preventDefault="printThis">保存图片</button>
            <button type="button" class="btn btn-outline-primary mx-0 mr-0" @click.preventDefault="shareTweet">分享至 Twitter</button>
          </div>
        </div>

        <div class="row col-12 mx-0 mt-3" ref="meme">
          <blockquote class="blockquote">
            <p v-html="content"></p>
          </blockquote>
        </div>
      </div>
    </div>

    <footer style="margin-top: -100px; width: 100%; line-height: 50px; background-color: #f5f5f5; text-align: center;">
      <div class="container">
        <a class="text-muted" href="https://github.com/disksing/sao-gen-gen">骚话生成器生成器@GitHub</a>
        <br/>
        <span class="text-muted">QQ交流群：481269635</span>
      </div>
    </footer>

    <script src="https://cdn.jsdelivr.net/npm/vue@2.6.14"></script>
    <script src="https://cdn.jsdelivr.net/npm/jquery@3.5.1/dist/jquery.slim.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/html2canvas@1.0.0-rc.7/dist/html2canvas.min.js"></script>
    <script src="https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/4.5.3/js/bootstrap.min.js"></script>

    <script>
      var config = {{.JSON}};

      var app = new Vue({
        el: "#app",
        data: {
          variables: config.variables,
          contentTpl: ""
        },
        created: function() {
          this.updateTpl();
        },
        methods: {
          async printThis() {
            console.log("printing..");
            const el = this.$refs.meme;

            const options = {
              type: "dataURL",
              scrollY: -window.scrollY
            };
            const printCanvas = await html2canvas(el, options);

            const link = document.createElement("a");
            link.setAttribute("download", "meme.png");
            link.setAttribute(
              "href",
              printCanvas
                .toDataURL("image/png")
                .replace("image/png", "image/octet-stream")
            );
            link.click();

            console.log("done");
          },
          shareTweet: function() {
            const content = this.getContent("plain") + '\\n' + '来自骚话生成器';
            const url = "https://twitter.com/intent/tweet?text=" + encodeURIComponent(content) + "&url=" + encodeURIComponent(window.location.href);
            window.open(url, '_blank', 'width=615,height=505');
          },
          updateTpl: function() {
            var templates = config.templates;
            for (k in templates) {
              var contents = templates[k];
              for (var i = contents.length-1; i > 0; i--) {
                var j = Math.floor(Math.random() * i);
                var tmp = contents[i];
                contents[i] = contents[j];
                contents[j] = tmp;
              }
            }

            var index = {};
            function expand(key, depth) {
              if (depth >= 16) {
                return "";
              }

              if (index[key] == null || index[key] >= templates[key].length) {
                index[key] = 0;
              }
              var t = templates[key][index[key]];
              index[key]++;

              out: while (true) {
                for (k in templates) {
                  if (t.search('{'+k+'}') != -1) {
                    t = t.replace('{'+k+'}', expand(k, depth+1));
                    continue out;
                  }
                }
                break;
              };
              return t;
            }
            this.contentTpl = expand('main', 0);
          },
          getContent: function(ty) {
            var t = this.contentTpl;
            for (i in this.variables) {
              var v = this.variables[i];
              var s = v.default;
              if (v.value != null && v.value.length > 0) {
                s = v.value;
              }
              if (ty == "html") {
                s = '<strong>'+s+'</strong>'
              }
              t = t.replaceAll('{'+v.id+'}', s);
            }
            if (ty == "html") {
              t = t.replaceAll('\n', '<br/>');
            }
            return t;
          },
          copy: function() {
            navigator.clipboard.writeText(this.getContent("plain")).then(function() {
            }, function() {
            });
          }
        },
        computed: {
          content: function() { return this.getContent("html"); }
        }
      });

    </script>
  </body>
</html>
`
