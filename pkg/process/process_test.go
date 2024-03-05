package process

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("ZeroReturnCode", func(t *testing.T) {
		t.Skip("TODO: test just hangs")

		stdoutReader, stdoutWriter := io.Pipe()
		defer func() {
			_ = stdoutReader.Close()
			_ = stdoutWriter.Close()
		}()

		stderrReader, stderrWriter := io.Pipe()
		defer func() {
			_ = stderrReader.Close()
			_ = stderrWriter.Close()
		}()

		p := Run(
			"/bin/bash",
			"echo 'first'; sleep 0.1; echo 'second'",
			nil,
			true,
			stdoutWriter,
			stderrWriter,
		)
		defer p.Close()

		err := p.Wait()
		require.NoError(t, err)
		require.Equal(t, 0, p.ReturnCode())

		out, err := io.ReadAll(stdoutReader)
		require.NoError(t, err)
		require.Equal(t, "first\nsecond\n", string(out))
	})

	t.Run("NonZeroReturnCode", func(t *testing.T) {
		t.Skip("TODO: test just hangs")

		stdoutReader, stdoutWriter := io.Pipe()
		defer func() {
			_ = stdoutReader.Close()
			_ = stdoutWriter.Close()
		}()

		stderrReader, stderrWriter := io.Pipe()
		defer func() {
			_ = stderrReader.Close()
			_ = stderrWriter.Close()
		}()

		p := Run(
			"/bin/bash",
			"echo 'first'; sleep 0.1; echo 'second'; false",
			nil,
			true,
			stdoutWriter,
			stderrWriter,
		)
		defer p.Close()
		err := p.Wait()
		require.Error(t, err)

		require.NotEqual(t, 0, p.ReturnCode())

		out, err := io.ReadAll(stdoutReader)
		require.NoError(t, err)
		require.Equal(t, "first\nsecond\n", string(out))
	})

	t.Run("NotACommand", func(t *testing.T) {
		t.Skip("TODO: test just hangs")

		stdoutReader, stdoutWriter := io.Pipe()
		defer func() {
			_ = stdoutReader.Close()
			_ = stdoutWriter.Close()
		}()

		stderrReader, stderrWriter := io.Pipe()
		defer func() {
			_ = stderrReader.Close()
			_ = stderrWriter.Close()
		}()

		p := Run(
			"/bin/bash",
			"not-a-command",
			nil,
			true,
			stdoutWriter,
			stderrWriter,
		)
		defer p.Close()
		err := p.Wait()
		require.Error(t, err)

		require.NotEqual(t, 0, p.ReturnCode())

		out, err := io.ReadAll(stdoutReader)
		require.NoError(t, err)
		require.Equal(t, "/bin/bash: not-a-command: command not found\n", string(out))
	})
}
