package dfa

import (
	"fmt"
	"strings"
)

const ALPHABET_SIZE int = 38
const OFFSET_MASK uint32 = 0x000FFFFF
const TERMINAL_BIT uint32 = 0x80000000
const GROW_SIZE = ALPHABET_SIZE * 28

type Tree struct {
	slots  []uint32
	offset int
}

func (t *Tree) grow() {
	if t.offset+ALPHABET_SIZE > len(t.slots) {
		temp := make([]uint32, len(t.slots)+GROW_SIZE)
		copy(temp, t.slots)
		t.slots = temp
	}
}

func NewTree() *Tree {
	return &Tree{slots: make([]uint32, GROW_SIZE), offset: 0}
}

func (t *Tree) AddPattern(pattern string) error {
	value, err := prepare_hostname(pattern)
	if err != nil {
		return err
	}

	off := 0
	last := len(value) - 1
	for p, c := range value {
		idx := char_to_index(c)
		fmt.Printf("Index is %d\n", idx)
		if t.slots[off+idx]&OFFSET_MASK == 0 {
			if p == last {
				t.slots[off+idx] |= TERMINAL_BIT
				fmt.Printf("Done\n")
			} else {
				t.grow()
				t.offset += ALPHABET_SIZE
				t.slots[off+idx] = (uint32(t.offset) / uint32(ALPHABET_SIZE)) & OFFSET_MASK
				off = t.offset
				fmt.Printf("Going to %d\n", off)
			}
		} else {
			off = int(t.slots[off+idx]&OFFSET_MASK) * ALPHABET_SIZE
		}
	}

	return nil
}

func (t *Tree) Print() {
	t.print_node(0, 0)
}

func (t *Tree) print_node(off int, level int) {
	for i := 0; i < ALPHABET_SIZE; i++ {
		next := int(t.slots[off+i]&OFFSET_MASK) * ALPHABET_SIZE
		if t.slots[off+i] != 0 {
			fmt.Printf("%s%c\n", strings.Repeat(" ", level), index_to_char(i))
			if next != 0 {
				t.print_node(next, level+1)
			}
		}
	}
}

func (t *Tree) Match(host string) bool {
	value, err := prepare_hostname(host)
	if err != nil {
		return false
	}

	off := 0
	last := len(value) - 1
	for p, c := range value {
		idx := char_to_index(c)
		if t.slots[off+idx]&TERMINAL_BIT != 0 {
			return true
		}
		if p == last {
			return false
		}
		if t.slots[off+idx]&OFFSET_MASK != 0 {
			off = int(t.slots[off+idx]&OFFSET_MASK) * ALPHABET_SIZE
		} else {
			return false // shouldn't happen
		}
	}
	return false
}

func char_to_index(c rune) int {
	if c >= 'A' && c <= 'Z' {
		return int(c) - 'A' // 0..25
	}
	if c >= 'a' && c <= 'z' {
		return int(c) - 'a' // 0..25
	}
	if c >= '0' && c <= '9' {
		return int(c) - '0' + 26 // 26..35
	}
	if c == '-' {
		return 36
	}
	if c == '.' {
		return 37
	}
	return -1
}

func index_to_char(index int) rune {
	if index >= 0 && index <= 25 {
		return rune('A' + index)
	}
	if index >= 26 && index <= 35 {
		return rune('0' + index)
	}
	if index == 36 {
		return '-'
	}
	if index == 37 {
		return '.'
	}
	return '?'
}

func reverse_string(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func prepare_hostname(host string) (string, error) {
	if host == "" {
		return "", nil
	}
	// remove leading and trailing unused characters
	host = strings.TrimSpace(host)
	// validate the host characters
	for _, c := range host {
		if char_to_index(c) < 0 {
			return "", fmt.Errorf("Invalid character")
		}
	}
	// reverse the symbols
	return reverse_string(host), nil
}
