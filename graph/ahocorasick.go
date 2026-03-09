// Package graph provides stringology and graph data structures for the SCS tool.
package graph

// AlphabetCapacity defines the byte-width alphabet for the automaton.
const AlphabetCapacity = 256

// Node defines a single transition state within the Aho-Corasick automaton.
// All fields use value types to ensure flat, GC-friendly memory layout.
type Node struct {
	// child maps byte values to subsequent node indices.
	// A fixed array guarantees O(1) transition lookups.
	child [AlphabetCapacity]int

	// fail points to the fallback node index upon a transition failure.
	fail int

	// output holds the identifiers of dictionary patterns terminating at this state.
	output []int

	// matchLengths caches the byte lengths of matched patterns.
	matchLengths []int
}

// AhoCorasick encapsulates a flat-memory, slice-based state machine.
type AhoCorasick struct {
	nodes []Node
}

// MatchRecord standardizes the boundary output of a search hit.
type MatchRecord struct {
	PatternID int
	Start     int
	End       int
}

// InitializeAutomaton creates an automaton with a pre-allocated root node.
// Index 0 is exclusively reserved as the structural root.
func InitializeAutomaton(prealloc int) *AhoCorasick {
	ac := &AhoCorasick{
		nodes: make([]Node, 1, prealloc+1),
	}
	return ac
}

// InsertPattern injects a dictionary string and its unique identifier into the trie.
func (ac *AhoCorasick) InsertPattern(pattern string, patternID int) {
	currentState := 0

	for i := 0; i < len(pattern); i++ {
		charIdx := pattern[i]

		if ac.nodes[currentState].child[charIdx] == 0 {
			allocatedNodeIdx := len(ac.nodes)
			ac.nodes = append(ac.nodes, Node{})
			ac.nodes[currentState].child[charIdx] = allocatedNodeIdx
		}

		currentState = ac.nodes[currentState].child[charIdx]
	}

	ac.nodes[currentState].output = append(ac.nodes[currentState].output, patternID)
	ac.nodes[currentState].matchLengths = append(ac.nodes[currentState].matchLengths, len(pattern))
}

// ComputeFailureLinks resolves fallback and output vectors via BFS traversal.
func (ac *AhoCorasick) ComputeFailureLinks() {
	queue := make([]int, 0)

	// Depth-1 states intrinsically fail back to the root.
	for c := 0; c < AlphabetCapacity; c++ {
		childIdx := ac.nodes[0].child[c]
		if childIdx != 0 {
			ac.nodes[childIdx].fail = 0
			queue = append(queue, childIdx)
		}
	}

	// BFS resolution across remaining depths.
	for len(queue) > 0 {
		currentState := queue[0]
		queue = queue[1:]

		for c := 0; c < AlphabetCapacity; c++ {
			childIdx := ac.nodes[currentState].child[c]
			if childIdx == 0 {
				continue
			}

			queue = append(queue, childIdx)

			// Traverse the failure trajectory of the parent.
			fallbackState := ac.nodes[currentState].fail
			for fallbackState != 0 && ac.nodes[fallbackState].child[c] == 0 {
				fallbackState = ac.nodes[fallbackState].fail
			}

			// Resolve the failure link for the child.
			ac.nodes[childIdx].fail = ac.nodes[fallbackState].child[c]

			// Prevent self-loops: a node must not fail back to itself.
			if ac.nodes[childIdx].fail == childIdx {
				ac.nodes[childIdx].fail = 0
			}

			// Inherit output links from the resolved failure state.
			resolvedFailIdx := ac.nodes[childIdx].fail
			ac.nodes[childIdx].output = append(
				ac.nodes[childIdx].output,
				ac.nodes[resolvedFailIdx].output...,
			)
			ac.nodes[childIdx].matchLengths = append(
				ac.nodes[childIdx].matchLengths,
				ac.nodes[resolvedFailIdx].matchLengths...,
			)
		}
	}
}

// Search stream-processes an input text against the active automaton.
// Returns all match records found during the traversal.
func (ac *AhoCorasick) Search(text string) []MatchRecord {
	var records []MatchRecord
	currentState := 0

	for i := 0; i < len(text); i++ {
		charIdx := text[i]

		// Regress through the failure chain upon a broken path.
		for currentState != 0 && ac.nodes[currentState].child[charIdx] == 0 {
			currentState = ac.nodes[currentState].fail
		}

		// Execute the definitive transition.
		currentState = ac.nodes[currentState].child[charIdx]

		// Yield pattern records if the state has dictionary mappings.
		for idx, patternID := range ac.nodes[currentState].output {
			length := ac.nodes[currentState].matchLengths[idx]
			records = append(records, MatchRecord{
				PatternID: patternID,
				Start:     i - length + 1,
				End:       i,
			})
		}
	}

	return records
}
