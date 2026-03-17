//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dir := filepath.Join("testdata")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	writeSyntheticJSON(dir)
	writeCL100kJSON(dir)
	writeO200kJSON(dir)
	writeP50kBaseJSON(dir)
	writeP50kEditJSON(dir)
	writeR50kJSON(dir)
	fmt.Println("Done.")
}

// tok builds a "<|name|>" special token safely (avoids Go's HTML escaping in JSON).
func tok(name string) string {
	return "<|" + name + "|>"
}

func writeSyntheticJSON(dir string) {
	m := map[string]interface{}{
		"pat_str": `\w+|\S`,
		"special_tokens": map[string]int{
			tok("synthetic_end"): 100,
		},
	}
	writeJSON(filepath.Join(dir, "synthetic.json"), m)
}

// r50k pat_str is shared by r50k_base, p50k_base, and p50k_edit.
const r50kPatStr = `'(?:[sdmt]|ll|ve|re)| ?\p{L}+| ?\p{N}+| ?[^\s\p{L}\p{N}]+|\s+$|\s+(?!\S)|\s`

func writeCL100kJSON(dir string) {
	pat := `(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\r\n\p{L}\p{N}]?\p{L}+|\p{N}{1,3}| ?[^\s\p{L}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+(?!\S)|\s+`
	m := map[string]interface{}{
		"pat_str": pat,
		"special_tokens": map[string]int{
			tok("endoftext"):   100257,
			tok("fim_prefix"):  100258,
			tok("fim_middle"):  100259,
			tok("fim_suffix"):  100260,
			tok("endofprompt"): 100276,
		},
	}
	writeJSON(filepath.Join(dir, "cl100k_base.json"), m)
}

func writeO200kJSON(dir string) {
	pat := `[^\r\n\p{L}\p{N}]?[\p{Lu}\p{Lt}\p{Lm}\p{Lo}\p{M}]*[\p{Ll}\p{Lm}\p{Lo}\p{M}]+(?i:'s|'t|'re|'ve|'m|'ll|'d)?` +
		`|[^\r\n\p{L}\p{N}]?[\p{Lu}\p{Lt}\p{Lm}\p{Lo}\p{M}]+[\p{Ll}\p{Lm}\p{Lo}\p{M}]*(?i:'s|'t|'re|'ve|'m|'ll|'d)?` +
		`|\p{N}{1,3}` +
		`| ?[^\s\p{L}\p{N}]+[\r\n/]*` +
		`|\s*[\r\n]+` +
		`|\s+(?!\S)` +
		`|\s+`
	m := map[string]interface{}{
		"pat_str": pat,
		"special_tokens": map[string]int{
			tok("endoftext"):   199999,
			tok("endofprompt"): 200018,
		},
	}
	writeJSON(filepath.Join(dir, "o200k_base.json"), m)
}

func writeP50kBaseJSON(dir string) {
	m := map[string]interface{}{
		"pat_str": r50kPatStr,
		"special_tokens": map[string]int{
			tok("endoftext"): 50256,
		},
	}
	writeJSON(filepath.Join(dir, "p50k_base.json"), m)
}

func writeP50kEditJSON(dir string) {
	m := map[string]interface{}{
		"pat_str": r50kPatStr,
		"special_tokens": map[string]int{
			tok("endoftext"):  50256,
			tok("fim_prefix"): 50281,
			tok("fim_middle"): 50282,
			tok("fim_suffix"): 50283,
		},
	}
	writeJSON(filepath.Join(dir, "p50k_edit.json"), m)
}

func writeR50kJSON(dir string) {
	m := map[string]interface{}{
		"pat_str": r50kPatStr,
		"special_tokens": map[string]int{
			tok("endoftext"): 50256,
		},
	}
	writeJSON(filepath.Join(dir, "r50k_base.json"), m)
}

func writeJSON(path string, v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal %s: %v\n", path, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("  Wrote %s\n", path)
}
