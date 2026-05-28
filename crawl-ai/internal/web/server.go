package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/yellowhama/musu-crawl-ai/internal/processor"
	"github.com/yuin/goldmark"
)

type Server struct {
	WikiDir string
	Port    int
	Project string
}

func NewServer(wikiDir string, port int, project string) *Server {
	return &Server{WikiDir: wikiDir, Port: port, Project: project}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/search", s.handleSearch)
	http.HandleFunc("/view", s.handleView)
	http.HandleFunc("/galaxy", s.handleGalaxy)
	http.HandleFunc("/api/graph", s.handleAPIGraph)

	fmt.Printf("🌐 musu-crawl dashboard v0.3.0 starting at http://localhost:%d\n", s.Port)
	fmt.Printf("🎯 Active Project Scope: %s\n", s.Project)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	indexFile := filepath.Join(s.WikiDir, "index.json")
	var allEntries []processor.IndexEntry

	if data, err := os.ReadFile(indexFile); err == nil {
		if uerr := json.Unmarshal(data, &allEntries); uerr != nil {
			fmt.Fprintf(os.Stderr, "warn: parse %s: %v\n", indexFile, uerr)
		}
	}

	var filtered []processor.IndexEntry
	projects := make(map[string]int)
	for _, e := range allEntries {
		projects[e.Project]++
		if s.Project == "all" || e.Project == s.Project {
			filtered = append(filtered, e)
		}
	}

	tmpl := template.Must(template.New("layout").Parse(layoutHTML))
	template.Must(tmpl.New("content").Parse(indexHTML))

	tmpl.Execute(w, map[string]interface{}{
		"Entries":    filtered,
		"Projects":   projects,
		"CurProject": s.Project,
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	queryStr := r.URL.Query().Get("q")
	blevePath := filepath.Join(s.WikiDir, "musu.bleve")

	var results []processor.IndexEntry

	if queryStr != "" {
		index, err := bleve.Open(blevePath)
		if err == nil {
			defer index.Close()

			// Apply project filter to search
			searchQuery := queryStr
			if s.Project != "all" {
				searchQuery = fmt.Sprintf("+project:%s %s", s.Project, queryStr)
			}

			query := bleve.NewQueryStringQuery(searchQuery)
			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Fields = []string{"title", "source", "id", "summary", "path", "project"}
			searchRes, err := index.Search(searchRequest)
			if err == nil {
				for _, hit := range searchRes.Hits {
					results = append(results, processor.IndexEntry{
						ID:      hit.ID,
						Title:   hit.Fields["title"].(string),
						Source:  hit.Fields["source"].(string),
						Summary: hit.Fields["summary"].(string),
						Path:    hit.Fields["path"].(string),
						Project: hit.Fields["project"].(string),
					})
				}
			}
		}
	}

	tmpl := template.Must(template.New("layout").Parse(layoutHTML))
	template.Must(tmpl.New("content").Parse(searchHTML))

	tmpl.Execute(w, map[string]interface{}{
		"Query":      queryStr,
		"Results":    results,
		"CurProject": s.Project,
	})
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	fullPath := filepath.Join(s.WikiDir, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "File not found", 404)
		return
	}

	content := string(data)
	if idx := strings.Index(content, "---"); idx == 0 {
		if nextIdx := strings.Index(content[3:], "---"); nextIdx != -1 {
			content = content[nextIdx+6:]
		}
	}

	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(content), &buf); err != nil {
		http.Error(w, "Markdown conversion failed", 500)
		return
	}

	tmpl := template.Must(template.New("layout").Parse(layoutHTML))
	template.Must(tmpl.New("content").Parse(viewHTML))

	tmpl.Execute(w, map[string]interface{}{
		"Title":      filepath.Base(path),
		"Content":    template.HTML(buf.String()),
		"CurProject": s.Project,
	})
}

func (s *Server) handleGalaxy(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("layout").Parse(layoutHTML))
	template.Must(tmpl.New("content").Parse(galaxyHTML))
	tmpl.Execute(w, map[string]interface{}{
		"CurProject": s.Project,
	})
}

func (s *Server) handleAPIGraph(w http.ResponseWriter, r *http.Request) {
	indexFile := filepath.Join(s.WikiDir, "index.json")
	var entries []processor.IndexEntry
	if data, err := os.ReadFile(indexFile); err == nil {
		if uerr := json.Unmarshal(data, &entries); uerr != nil {
			fmt.Fprintf(os.Stderr, "warn: parse %s: %v\n", indexFile, uerr)
		}
	}

	type Node struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Group string `json:"group"` // source or project
	}
	type Link struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Value  int    `json:"value"`
	}
	type Graph struct {
		Nodes []Node `json:"nodes"`
		Links []Link `json:"links"`
	}

	graph := Graph{
		Nodes: []Node{},
		Links: []Link{},
	}

	for _, e := range entries {
		if s.Project != "all" && e.Project != s.Project {
			continue
		}
		graph.Nodes = append(graph.Nodes, Node{ID: e.ID, Name: e.Title, Group: e.Source})

		// Parse content for WikiLinks [[ID]] to build edges
		fullPath := filepath.Join(s.WikiDir, e.Path)
		if data, err := os.ReadFile(fullPath); err == nil {
			content := string(data)
			re := regexp.MustCompile(`\[\[(.*?)\]\]`)
			matches := re.FindAllStringSubmatch(content, -1)
			for _, m := range matches {
				targetID := m[1]
				// Basic check if target exists
				graph.Links = append(graph.Links, Link{Source: e.ID, Target: targetID, Value: 1})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

// --- Embedded Templates ---

const layoutHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MUSU AI Brain Portal</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        'musu-accent': '#ffa602',
                        'musu-deep': '#432c1c',
                        'musu-cream': '#FDF8F1',
                        'musu-dark': '#2D1E12',
                    }
                }
            }
        }
    </script>
    <style>
        .prose h1, .prose h2, .prose h3 { color: #432c1c; font-weight: bold; }
        .prose h1 { font-size: 2.25rem; margin-bottom: 1.5rem; border-bottom: 2px solid #ffa602; padding-bottom: 0.5rem; }
        .prose h2 { font-size: 1.75rem; margin-top: 2rem; margin-bottom: 1rem; }
        .prose p { margin-bottom: 1.25rem; line-height: 1.7; color: #333; }
        .prose a { color: #ffa602; font-weight: bold; text-decoration: none; border-bottom: 1px solid transparent; transition: 0.2s; }
        .prose a:hover { border-bottom-color: #ffa602; }
        .prose code { background: #f3f4f6; padding: 0.2rem 0.4rem; rounded: 0.25rem; font-size: 0.9em; }
        .prose pre { background: #1f2937; color: #f9fafb; padding: 1.5rem; rounded: 0.75rem; overflow-x: auto; margin: 1.5rem 0; }
        .prose blockquote { border-left: 4px solid #ffa602; padding-left: 1.5rem; italic: true; color: #4b5563; margin: 1.5rem 0; }
        
        /* Galaxy Background */
        .galaxy-bg { background: radial-gradient(circle at center, #432c1c 0%, #1a130d 100%); }
    </style>
</head>
<body class="bg-musu-cream text-musu-deep min-h-screen flex flex-col font-sans">
    <nav class="bg-musu-deep text-white p-4 shadow-2xl sticky top-0 z-50">
        <div class="container mx-auto flex justify-between items-center">
            <div class="flex items-center gap-8">
                <a href="/" class="text-3xl font-black tracking-tighter text-musu-accent flex items-center gap-2">
                    <span class="bg-musu-accent text-musu-deep px-2 py-0.5 rounded">M</span> USU
                </a>
                <div class="hidden md:flex gap-6 text-sm font-bold uppercase tracking-widest">
                    <a href="/" class="hover:text-musu-accent transition">Dashboard</a>
                    <a href="/galaxy" class="hover:text-musu-accent transition">Knowledge Galaxy</a>
                </div>
            </div>
            <form action="/search" method="GET" class="flex gap-2">
                <div class="relative group">
                    <input type="text" name="q" placeholder="Query Brain..." class="bg-musu-dark border border-musu-accent/30 rounded-full px-5 py-2 text-white focus:ring-2 focus:ring-musu-accent outline-none w-64 transition-all">
                    <div class="absolute right-3 top-2.5 opacity-40 group-focus-within:opacity-100">🔍</div>
                </div>
            </form>
        </div>
    </nav>

    <main class="flex-grow container mx-auto py-10 px-6">
        {{template "content" .}}
    </main>

    <footer class="bg-musu-deep text-white/40 py-10 border-t border-musu-accent/10">
        <div class="container mx-auto px-6 flex justify-between items-center">
            <div class="text-sm">
                &copy; 2026 musu-crawl-ai v0.3.0 • <span class="text-musu-accent/60">Autonomous Researcher</span>
            </div>
            <div class="flex gap-4 font-mono text-xs">
                <span>PROJECT: {{.CurProject}}</span>
                <span class="text-green-500">OLLAMA: READY</span>
            </div>
        </div>
    </footer>
</body>
</html>
`

const indexHTML = `
<div class="flex flex-col lg:flex-row gap-10">
    <!-- Sidebar -->
    <aside class="lg:w-1/4 space-y-8">
        <div class="bg-white p-6 rounded-2xl shadow-sm border border-musu-deep/5">
            <h3 class="text-xs font-black uppercase text-gray-400 mb-4 tracking-widest">Active Projects</h3>
            <nav class="flex flex-col gap-2">
                <a href="/" class="px-4 py-2 rounded-lg {{if eq .CurProject "all"}}bg-musu-accent text-musu-deep font-bold{{else}}hover:bg-musu-cream{{end}} transition">All Knowledge</a>
                {{range $name, $count := .Projects}}
                <a href="/?project={{$name}}" class="flex justify-between items-center px-4 py-2 rounded-lg hover:bg-musu-cream transition group">
                    <span>{{$name}}</span>
                    <span class="text-xs bg-gray-100 px-2 py-0.5 rounded group-hover:bg-musu-accent transition">{{$count}}</span>
                </a>
                {{end}}
            </nav>
        </div>
        
        <div class="bg-musu-deep p-6 rounded-2xl shadow-xl text-white">
            <h3 class="text-musu-accent font-bold mb-2">Agent Status</h3>
            <p class="text-xs text-white/60 leading-relaxed mb-4">The research agent is currently idle. Ready for the next knowledge mission.</p>
            <button class="w-full bg-musu-accent text-musu-deep font-black py-2 rounded-xl text-sm hover:scale-105 transition">NEW RESEARCH</button>
        </div>
    </aside>

    <!-- Main Content -->
    <div class="lg:w-3/4">
        <div class="flex justify-between items-end mb-8">
            <div>
                <p class="text-musu-accent font-mono text-sm uppercase tracking-widest mb-1">Knowledge Repository</p>
                <h1 class="text-4xl font-black">AI Brain Archive</h1>
            </div>
            <div class="text-right">
                <span class="text-4xl font-light text-gray-300">{{len .Entries}}</span>
                <span class="text-xs font-bold text-gray-400 block uppercase">Docs Indexed</span>
            </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            {{range .Entries}}
            <a href="/view?path={{.Path}}" class="bg-white p-6 rounded-3xl shadow-sm border border-musu-deep/5 hover:shadow-2xl hover:-translate-y-1 transition duration-300 group">
                <div class="flex justify-between items-start mb-4">
                    <span class="bg-musu-cream text-musu-deep text-[10px] font-black px-2 py-1 rounded uppercase tracking-tighter">{{.Source}}</span>
                    <span class="text-[10px] font-bold text-gray-300 uppercase">{{.Date}}</span>
                </div>
                <h2 class="text-xl font-bold mb-3 group-hover:text-musu-accent transition">{{.Title}}</h2>
                <p class="text-gray-500 text-sm line-clamp-3 leading-relaxed mb-6">{{.Summary}}</p>
                <div class="flex flex-wrap gap-2">
                    {{range .Tags}}
                    <span class="text-[9px] font-bold text-gray-400 border border-gray-100 px-2 py-0.5 rounded-full">#{{.}}</span>
                    {{end}}
                </div>
            </a>
            {{end}}
        </div>
    </div>
</div>
`

const searchHTML = `
<div class="max-w-4xl mx-auto">
    <div class="mb-12">
        <h1 class="text-4xl font-black mb-2 tracking-tight">Search Results</h1>
        <p class="text-gray-400">Found {{len .Results}} intelligence nodes for <span class="text-musu-accent font-bold">"{{.Query}}"</span></p>
    </div>

    <div class="space-y-6">
        {{range .Results}}
        <div class="bg-white p-8 rounded-3xl shadow-sm border-l-8 border-musu-accent hover:shadow-xl transition group">
            <div class="flex gap-4 items-center mb-3">
                <span class="text-xs font-black uppercase tracking-widest text-musu-accent">{{.Source}}</span>
                <span class="w-1 h-1 bg-gray-200 rounded-full"></span>
                <span class="text-xs font-bold text-gray-400 uppercase">PROJECT: {{.Project}}</span>
            </div>
            <h2 class="text-2xl font-bold mb-3 group-hover:text-musu-accent transition">{{.Title}}</h2>
            <p class="text-gray-600 mb-6 leading-relaxed">{{.Summary}}</p>
            <a href="/view?path={{.Path}}" class="inline-flex items-center gap-2 text-musu-deep font-black text-sm uppercase tracking-tighter hover:gap-4 transition-all">
                Access Intelligence Node <span>→</span>
            </a>
        </div>
        {{else}}
        <div class="text-center py-20 bg-white rounded-3xl border-2 border-dashed border-gray-100">
            <p class="text-gray-400 italic">The brain found no relevant connections for this query.</p>
        </div>
        {{end}}
    </div>
</div>
`

const viewHTML = `
<div class="max-w-4xl mx-auto">
    <nav class="flex justify-between items-center mb-10">
        <a href="/" class="text-sm font-bold text-musu-deep/40 hover:text-musu-accent transition uppercase tracking-widest flex items-center gap-2">
            <span>←</span> Back to Dashboard
        </a>
        <div class="flex gap-4">
             <button class="bg-musu-deep text-white px-4 py-2 rounded-xl text-xs font-bold uppercase hover:bg-musu-accent hover:text-musu-deep transition">Export PDF</button>
             <button class="bg-musu-accent text-musu-deep px-4 py-2 rounded-xl text-xs font-bold uppercase hover:opacity-80 transition">Cite Node</button>
        </div>
    </nav>
    
    <article class="bg-white p-12 md:p-16 rounded-[3rem] shadow-2xl border border-musu-deep/5 prose prose-slate max-w-none">
        {{.Content}}
    </article>
    
    <section class="mt-12 p-8 bg-musu-deep rounded-3xl text-white">
        <h3 class="text-musu-accent font-black uppercase text-xs tracking-widest mb-4">Agent Reflection</h3>
        <p class="text-sm text-white/70 leading-relaxed italic italic">"This node was harvested during the recursive exploration of {{.Title}}. It provides critical evidence for understanding the core concepts of this project."</p>
    </section>
</div>
`

const galaxyHTML = `
<div class="fixed inset-0 top-16 z-0 galaxy-bg"></div>

<div class="relative z-10">
    <div class="flex justify-between items-start pointer-events-none">
        <div class="bg-musu-deep/80 backdrop-blur-xl p-8 rounded-br-3xl border-b border-r border-musu-accent/20">
            <h1 class="text-4xl font-black text-white mb-2">Knowledge Galaxy</h1>
            <p class="text-musu-accent font-mono text-sm tracking-widest uppercase">Visualizing connections in the AI Brain</p>
        </div>
        
        <div class="p-8 text-right bg-musu-deep/80 backdrop-blur-xl rounded-bl-3xl border-b border-l border-musu-accent/20">
            <div class="text-white/40 text-[10px] font-bold uppercase tracking-widest mb-2">Galaxy Legend</div>
            <div class="flex gap-4 text-[10px] font-bold">
                <span class="flex items-center gap-2 text-white"><span class="w-2 h-2 rounded-full bg-blue-400"></span> Papers</span>
                <span class="flex items-center gap-2 text-white"><span class="w-2 h-2 rounded-full bg-red-400"></span> GitHub</span>
                <span class="flex items-center gap-2 text-white"><span class="w-2 h-2 rounded-full bg-yellow-400"></span> YouTube</span>
            </div>
        </div>
    </div>

    <div id="galaxy-map" class="w-full h-[70vh] cursor-move"></div>
</div>

<script>
    const width = window.innerWidth;
    const height = window.innerHeight;

    const svg = d3.select("#galaxy-map")
        .append("svg")
        .attr("width", "100%")
        .attr("height", "100%")
        .call(d3.zoom().on("zoom", function (event) {
            container.attr("transform", event.transform);
        }))
        .append("g");

    const container = svg.append("g");

    d3.json("/api/graph").then(data => {
        const simulation = d3.forceSimulation(data.nodes)
            .force("link", d3.forceLink(data.links).id(d => d.id).distance(150))
            .force("charge", d3.forceManyBody().strength(-300))
            .force("center", d3.forceCenter(width / 2, height / 2.5));

        const link = container.append("g")
            .attr("stroke", "#ffa602")
            .attr("stroke-opacity", 0.3)
            .selectAll("line")
            .data(data.links)
            .join("line")
            .attr("stroke-width", d => Math.sqrt(d.value) * 2);

        const node = container.append("g")
            .selectAll("g")
            .data(data.nodes)
            .join("g")
            .attr("class", "cursor-pointer")
            .on("click", (e, d) => {
                // Find path from data
                const entry = data.nodes.find(n => n.id === d.id);
                // We'll just search for the node title in index
                window.location.href = "/search?q=" + d.name;
            });

        node.append("circle")
            .attr("r", 12)
            .attr("fill", d => {
                if (d.group === "papers") return "#60a5fa";
                if (d.group === "github") return "#f87171";
                if (d.group === "youtube") return "#fbbf24";
                return "#ffa602";
            })
            .attr("stroke", "#fff")
            .attr("stroke-width", 2)
            .attr("class", "hover:scale-150 transition-transform duration-300");

        node.append("text")
            .attr("x", 16)
            .attr("y", 4)
            .text(d => d.name)
            .attr("fill", "white")
            .attr("font-size", "10px")
            .attr("font-weight", "bold")
            .attr("style", "text-shadow: 0 2px 4px rgba(0,0,0,0.5)");

        simulation.on("tick", () => {
            link
                .attr("x1", d => d.source.x)
                .attr("y1", d => d.source.y)
                .attr("x2", d => d.target.x)
                .attr("y2", d => d.target.y);

            node
                .attr("transform", d => ` + "`" + `translate(${d.x},${d.y})` + "`" + `);
        });
    });
</script>
`
