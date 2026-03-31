package blockeditor

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

//go:embed web/*
var webFiles embed.FS

// Server serves the block editor web UI
type Server struct {
	analyzer *Analyzer
	port     int
}

// NewServer creates a new block editor web server
func NewServer(analyzer *Analyzer, port int) *Server {
	return &Server{
		analyzer: analyzer,
		port:     port,
	}
}

// Start starts the web server
func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/diskmap", s.handleDiskMap)
	http.HandleFunc("/api/blocks", s.handleBlocks)
	http.HandleFunc("/api/block/", s.handleBlock)
	http.HandleFunc("/api/file/", s.handleFile)
	http.HandleFunc("/api/search", s.handleSearch)
	http.HandleFunc("/api/statistics", s.handleStatistics)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("🌐 Block Editor UI available at: http://localhost%s", addr)
	log.Printf("📊 Analyzing %d blocks from: %s", s.analyzer.diskMap.TotalBlocks, s.analyzer.diskMap.ImagePath)

	return http.ListenAndServe(addr, nil)
}

// handleIndex serves the main HTML page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "Failed to load UI", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

// handleDiskMap returns the disk map metadata
func (s *Server) handleDiskMap(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"image_path":    s.analyzer.diskMap.ImagePath,
		"total_size":    s.analyzer.diskMap.TotalSize,
		"block_size":    s.analyzer.diskMap.BlockSize,
		"total_blocks":  s.analyzer.diskMap.TotalBlocks,
		"filesystem":    s.analyzer.diskMap.Filesystem,
		"volume_label":  s.analyzer.diskMap.VolumeLabel,
		"analysis_date": s.analyzer.diskMap.AnalysisDate,
		"statistics":    s.analyzer.diskMap.Statistics,
		"color_scheme":  DefaultColorScheme(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBlocks returns block summaries for a range
func (s *Server) handleBlocks(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	start := int64(0)
	end := s.analyzer.diskMap.TotalBlocks

	if startStr != "" {
		if val, err := strconv.ParseInt(startStr, 10, 64); err == nil {
			start = val
		}
	}

	if endStr != "" {
		if val, err := strconv.ParseInt(endStr, 10, 64); err == nil {
			end = val
		}
	}

	// Limit range to prevent memory issues
	maxRange := int64(100000) // 100K blocks at a time
	if end-start > maxRange {
		end = start + maxRange
	}

	summaries, err := s.analyzer.GetBlockSummaries(start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"start":  start,
		"end":    end,
		"count":  len(summaries),
		"blocks": summaries,
	})
}

// handleBlock returns detailed information about a specific block
func (s *Server) handleBlock(w http.ResponseWriter, r *http.Request) {
	// Extract block index from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/block/")
	index, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid block index", http.StatusBadRequest)
		return
	}

	block, err := s.analyzer.GetBlock(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(block)
}

// handleFile returns information about a file and its blocks
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	// Extract file ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/file/")
	fileID := path

	file, exists := s.analyzer.diskMap.Files[fileID]
	if !exists {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Get all blocks for this file
	blocks := make([]*Block, 0, len(file.Blocks))
	for _, blockIndex := range file.Blocks {
		if block, err := s.analyzer.GetBlock(blockIndex); err == nil {
			blocks = append(blocks, block)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"file":   file,
		"blocks": blocks,
	})
}

// handleSearch searches for blocks matching criteria
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var query SearchQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid search query", http.StatusBadRequest)
		return
	}

	results := s.analyzer.SearchBlocks(&query)

	// Convert to summaries for efficiency
	summaries := make([]*BlockSummary, len(results))
	for i, block := range results {
		summaries[i] = &BlockSummary{
			Index:   block.Index,
			Status:  block.Status,
			Type:    block.Type,
			FileID:  block.FileID,
			IsZero:  block.IsZero,
			Entropy: block.Entropy,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":   len(summaries),
		"results": summaries,
	})
}

// handleStatistics returns aggregate statistics
func (s *Server) handleStatistics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.analyzer.diskMap.Statistics)
}
