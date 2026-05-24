package guacamole

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Instruction represents a Guacamole protocol instruction.
// Format: length.field1,length.field2,...,length.fieldN;
type Instruction struct {
	Op   string
	Args []string
}

// WriteInstruction encodes an instruction and writes it to the writer.
func WriteInstruction(w io.Writer, op string, args ...string) error {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, encodeField(op))
	for _, arg := range args {
		parts = append(parts, encodeField(arg))
	}
	_, err := io.WriteString(w, strings.Join(parts, ",")+";")
	return err
}

func encodeField(s string) string {
	return fmt.Sprintf("%d.%s", len(s), s)
}

// ReadInstruction reads one instruction from the reader using buffered I/O.
func ReadInstruction(r io.Reader) (*Instruction, error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}

	line, err := br.ReadBytes(';')
	if err != nil {
		return nil, err
	}
	// Remove trailing ';'
	line = line[:len(line)-1]

	raw := string(line)
	if raw == "" {
		return nil, fmt.Errorf("empty instruction")
	}

	// Parse fields: "5.abc,3.def,"
	fields := strings.Split(raw, ",")
	instr := &Instruction{}
	for i, field := range fields {
		dotIdx := strings.Index(field, ".")
		if dotIdx < 0 {
			return nil, fmt.Errorf("malformed field %d: missing length prefix in %q", i, field)
		}
		length, err := strconv.Atoi(field[:dotIdx])
		if err != nil {
			return nil, fmt.Errorf("malformed field %d: invalid length in %q: %w", i, field, err)
		}
		value := field[dotIdx+1:]
		if len(value) > length {
			value = value[:length]
		}
		if i == 0 {
			instr.Op = value
		} else {
			instr.Args = append(instr.Args, value)
		}
	}
	return instr, nil
}
