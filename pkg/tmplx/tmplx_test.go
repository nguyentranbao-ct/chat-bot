package tmplx

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFields(t *testing.T) {
	t.Parallel()
	t.Run("single field", func(t *testing.T) {
		fields := ExtractFields("Hello, {{.Name}}!")
		expected := []string{"Name"}
		assert.Equal(t, expected, fields)
	})

	t.Run("multiple fields", func(t *testing.T) {
		fields := ExtractFields("Hello, {{.Name}}! Your age is {{.Age}}.")
		expected := []string{"Name", "Age"}
		assert.Equal(t, expected, fields)
	})

	t.Run("no fields", func(t *testing.T) {
		fields := ExtractFields("Hello, world!")
		expected := []string{}
		assert.Equal(t, expected, fields)
	})

	t.Run("field in function", func(t *testing.T) {
		fields := ExtractFields(`Hello, {{printf "%s" .Name}}!`)
		expected := []string{"Name"}
		assert.Equal(t, expected, fields)
	})

	t.Run("field in conditional", func(t *testing.T) {
		fields := ExtractFields(`{{if .Name}}Hello, {{.Name}}!{{end}}`)
		expected := []string{"Name"}
		assert.Equal(t, expected, fields)
	})

	t.Run("multiple fields in range", func(t *testing.T) {
		fields := ExtractFields(`{{range .People}}{{.Name}} is {{.Age}} years old.{{end}}`)
		expected := []string{"People", "Name", "Age"}
		assert.Equal(t, expected, fields)
	})

	t.Run("fields in with", func(t *testing.T) {
		fields := ExtractFields(`{{with .Person}}{{.Name}} is {{.Age}} years old.{{end}}`)
		expected := []string{"Person", "Name", "Age"}
		assert.Equal(t, expected, fields)
	})

	t.Run("field in if-else", func(t *testing.T) {
		fields := ExtractFields(`{{if .Name}}Hello, {{.Name}}!{{else}}Hello, stranger!{{end}}`)
		expected := []string{"Name"}
		assert.Equal(t, expected, fields)
	})

	t.Run("field in pipe function", func(t *testing.T) {
		fields := ExtractFields(`Hello, {{.Name | printf "%s"}}!`)
		expected := []string{"Name"}
		assert.Equal(t, expected, fields)
	})

	t.Run("multiple fields in if-else and range", func(t *testing.T) {
		fields := ExtractFields(`{{if .People}}{{range .People}}{{.Name}} is {{.Age}} years old.{{end}}{{else}}No people found.{{end}}`)
		expected := []string{"People", "Name", "Age"}
		assert.Equal(t, expected, fields)
	})

	t.Run("fields in with and pipe function", func(t *testing.T) {
		fields := ExtractFields(`{{with .Person}}{{.Name | printf "%s"}} is {{.Age}} years old.{{end}}`)
		expected := []string{"Person", "Name", "Age"}
		assert.Equal(t, expected, fields)
	})
}

func TestEncodeUrlQuery(t *testing.T) {
	t.Parallel()
	t.Run("encode url query", func(t *testing.T) {
		template := `{{ encodeUrlQuery "veh_inspected" "3" "cg" "2010" "region_v2" "13000" "st" "s,k" "carbrand" "18" }}`
		want := "carbrand=18&cg=2010&region_v2=13000&st=s%2Ck&veh_inspected=3"

		buf, err := MustParse("", template).Render(nil)
		require.NoError(t, err)
		got := strings.TrimSpace(buf.String())
		assert.Equal(t, want, got)
	})
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("with template func", func(t *testing.T) {
		tmpl, err := Parse("test", `{{custom}}`,
			WithTemplateFunc("custom", func() string { return "custom" }))
		require.NoError(t, err)

		buf, err := tmpl.Render(nil)
		require.NoError(t, err)
		assert.Equal(t, "custom", strings.TrimSpace(buf.String()))
	})

	t.Run("with validation", func(t *testing.T) {
		testData := map[string]string{"name": "test"}
		validateFn := func(buf *bytes.Buffer) error {
			if buf.String() != "test" {
				return fmt.Errorf("expected 'test', got '%s'", buf.String())
			}
			return nil
		}

		tmpl, err := Parse("test", `{{.name}}`, WithValidate(testData, validateFn))
		require.NoError(t, err)

		buf, err := tmpl.Render(testData)
		require.NoError(t, err)
		assert.Equal(t, "test", buf.String())
	})

	t.Run("merge with default funcs", func(t *testing.T) {
		tmpl, err := Parse("test", `{{custom}} {{quote "test"}}`,
			WithTemplateFunc("custom", func() string { return "custom" }))
		require.NoError(t, err)

		buf, err := tmpl.Render(nil)
		require.NoError(t, err)
		assert.Equal(t, `custom "test"`, strings.TrimSpace(buf.String()))
	})
}

func TestCustomFunctions(t *testing.T) {
	t.Parallel()

	t.Run("hasSuffix function", func(t *testing.T) {
		template := `{{if hasSuffix .text ".jpg"}}is image{{else}}not image{{end}}`
		data := map[string]any{
			"text": "photo.jpg",
		}

		tmpl := MustParse("", template)
		buf, err := tmpl.Render(data)
		require.NoError(t, err)
		assert.Equal(t, "is image", strings.TrimSpace(buf.String()))
	})

	t.Run("hasPrefix function", func(t *testing.T) {
		template := `{{if hasPrefix .text "https://"}}is secure{{else}}not secure{{end}}`
		data := map[string]any{
			"text": "https://example.com",
		}

		tmpl := MustParse("", template)
		buf, err := tmpl.Render(data)
		require.NoError(t, err)
		assert.Equal(t, "is secure", strings.TrimSpace(buf.String()))
	})

	t.Run("default function", func(t *testing.T) {
		template := `{{default "anonymous" .name}}`

		t.Run("with empty value", func(t *testing.T) {
			data := map[string]any{"name": ""}
			tmpl := MustParse("", template)
			buf, err := tmpl.Render(data)
			require.NoError(t, err)
			assert.Equal(t, "anonymous", strings.TrimSpace(buf.String()))
		})

		t.Run("with non-empty value", func(t *testing.T) {
			data := map[string]any{"name": "john"}
			tmpl := MustParse("", template)
			buf, err := tmpl.Render(data)
			require.NoError(t, err)
			assert.Equal(t, "john", strings.TrimSpace(buf.String()))
		})
	})

	t.Run("regexMatch function", func(t *testing.T) {
		template := `{{if regexMatch .text "^[0-9]+$"}}is number{{else}}not number{{end}}`

		t.Run("matching case", func(t *testing.T) {
			data := map[string]any{"text": "12345"}
			tmpl := MustParse("", template)
			buf, err := tmpl.Render(data)
			require.NoError(t, err)
			assert.Equal(t, "is number", strings.TrimSpace(buf.String()))
		})

		t.Run("non-matching case", func(t *testing.T) {
			data := map[string]any{"text": "abc123"}
			tmpl := MustParse("", template)
			buf, err := tmpl.Render(data)
			require.NoError(t, err)
			assert.Equal(t, "not number", strings.TrimSpace(buf.String()))
		})
	})

	t.Run("jsonGet function", func(t *testing.T) {
		template := `{{jsonGet "user.name" .json}}`
		data := map[string]any{
			"json": `{"user":{"name":"john","age":30}}`,
		}

		tmpl := MustParse("", template)
		buf, err := tmpl.Render(data)
		require.NoError(t, err)
		assert.Equal(t, "john", strings.TrimSpace(buf.String()))
	})
}

func TestTemplateValidation(t *testing.T) {
	t.Parallel()

	t.Run("successful validation", func(t *testing.T) {
		testData := map[string]any{
			"name": "john",
			"age":  30,
		}

		validateFn := func(buf *bytes.Buffer) error {
			if !strings.Contains(buf.String(), "john") {
				return fmt.Errorf("expected name 'john' in output")
			}
			if !strings.Contains(buf.String(), "30") {
				return fmt.Errorf("expected age '30' in output")
			}
			return nil
		}

		tmpl, err := Parse("test", `Name: {{.name}}, Age: {{.age}}`, WithValidate(testData, validateFn))
		require.NoError(t, err)

		buf, err := tmpl.Render(testData)
		require.NoError(t, err)
		assert.Equal(t, "Name: john, Age: 30", buf.String())
	})

	t.Run("failed validation", func(t *testing.T) {
		testData := map[string]any{
			"name": "invalid",
		}

		validateFn := func(buf *bytes.Buffer) error {
			if !strings.Contains(buf.String(), "john") {
				return fmt.Errorf("expected name 'john' in output")
			}
			return nil
		}

		_, err := Parse("test", `Name: {{.name}}`, WithValidate(testData, validateFn))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected name 'john' in output")
	})
}

func TestTemplateRenderError(t *testing.T) {
	t.Parallel()

	t.Run("invalid template syntax", func(t *testing.T) {
		template := `Hello {{.name`
		_, err := Parse("test", template)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrParseTemplate)
	})

	t.Run("no missing required field", func(t *testing.T) {
		template := `Hello {{.name}}`
		tmpl := MustParse("test", template)

		text, err := tmpl.Render(map[string]string{})
		require.NoError(t, err)
		assert.Equal(t, "Hello ", text.String())
	})
}

func TestTemplateParseError(t *testing.T) {
	t.Parallel()

	t.Run("invalid template syntax", func(t *testing.T) {
		_, err := Parse("test", `Hello {{.name`)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrParseTemplate)
	})

	t.Run("invalid function", func(t *testing.T) {
		_, err := Parse("test", `Hello {{.name | invalidFunc}}`)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrParseTemplate)
	})
}
