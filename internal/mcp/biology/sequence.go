package biology

import (
	"fmt"
	"strings"
	"unicode"
)

func maxAnalyzeLength() int {
	return biologySettings().MaxAnalyzeLengthOrDefault()
}

func maxFoldLength() int {
	return biologySettings().MaxFoldLengthOrDefault()
}

// normalizeSequence strips FASTA headers/whitespace and uppercases.
func normalizeSequence(raw string) string {
	var b strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ">") || strings.HasPrefix(line, ";") {
			continue
		}
		b.WriteString(line)
	}
	return strings.ToUpper(strings.TrimSpace(b.String()))
}

type seqKind int

const (
	seqUnknown seqKind = iota
	seqDNA
	seqRNA
	seqProtein
)

func classifySequence(seq string) seqKind {
	if seq == "" {
		return seqUnknown
	}
	dna, rna, protein, invalid := 0, 0, 0, 0
	for _, r := range seq {
		switch r {
		case 'A', 'C', 'G', 'T':
			dna++
		case 'U':
			rna++
		case 'N':
			// ambiguous — count toward all
			dna++
			rna++
		case '*':
			protein++
		default:
			if strings.ContainsRune("ACDEFGHIKLMNPQRSTVWY", r) {
				protein++
			} else {
				invalid++
			}
		}
	}
	if invalid > 0 && invalid > len(seq)/10 {
		return seqUnknown
	}
	if protein > 0 && protein >= dna && protein >= rna {
		return seqProtein
	}
	if rna > dna && rna > 0 {
		return seqRNA
	}
	if dna > 0 {
		return seqDNA
	}
	return seqUnknown
}

func reverseComplementDNA(seq string) string {
	complement := map[byte]byte{
		'A': 'T', 'T': 'A', 'C': 'G', 'G': 'C', 'N': 'N',
	}
	out := make([]byte, len(seq))
	for i := len(seq) - 1; i >= 0; i-- {
		c := seq[i]
		if v, ok := complement[c]; ok {
			out[len(seq)-1-i] = v
		} else {
			out[len(seq)-1-i] = c
		}
	}
	return string(out)
}

func analyzeSequenceText(raw string) (string, error) {
	seq := normalizeSequence(raw)
	if seq == "" {
		return "", fmt.Errorf("no sequence found (paste FASTA or raw sequence)")
	}
	maxLen := maxAnalyzeLength()
	truncated := false
	if len(seq) > maxLen {
		seq = seq[:maxLen]
		truncated = true
	}

	kind := classifySequence(seq)
	var kindStr string
	switch kind {
	case seqDNA:
		kindStr = "DNA"
	case seqRNA:
		kindStr = "RNA"
	case seqProtein:
		kindStr = "protein"
	default:
		kindStr = "unknown/ambiguous"
	}

	var invalid []string
	for i, r := range seq {
		if kind == seqProtein {
			if !unicode.IsLetter(r) && r != '*' {
				invalid = append(invalid, fmt.Sprintf("pos %d: %q", i+1, r))
			}
		} else if kind == seqDNA || kind == seqRNA {
			allowed := "ACGTN"
			if kind == seqRNA {
				allowed = "ACGUN"
			}
			if !strings.ContainsRune(allowed, r) {
				invalid = append(invalid, fmt.Sprintf("pos %d: %q", i+1, r))
			}
		}
		if len(invalid) >= 5 {
			break
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Sequence analysis (in silico)\n")
	fmt.Fprintf(&b, "Type: %s\n", kindStr)
	fmt.Fprintf(&b, "Length: %d residues/bases\n", len(seq))
	if truncated {
		fmt.Fprintf(&b, "Note: truncated to %d characters for analysis\n", maxLen)
	}
	if len(invalid) > 0 {
		fmt.Fprintf(&b, "Invalid characters (sample): %s\n", strings.Join(invalid, ", "))
	}
	if kind == seqDNA {
		rc := reverseComplementDNA(seq)
		if len(rc) <= 120 {
			fmt.Fprintf(&b, "Reverse complement: %s\n", rc)
		} else {
			fmt.Fprintf(&b, "Reverse complement (first 60): %s...\n", rc[:60])
		}
	}
	if kind == seqProtein && len(seq) <= maxFoldLength() {
		fmt.Fprintf(&b, "Eligible for fold_protein tool (length <= %d aa)\n", maxFoldLength())
	} else if kind == seqProtein {
		fmt.Fprintf(&b, "Too long for fold_protein (max %d aa); analyze or truncate first\n", maxFoldLength())
	}
	fmt.Fprintf(&b, "\nDisclaimer: research/education only; not clinical advice.\n")
	return b.String(), nil
}
