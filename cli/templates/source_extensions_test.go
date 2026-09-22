package templates

import "testing"

func TestIsSourceExtension(t *testing.T) {
	t.Parallel()
	tests := []struct {
		ext  string
		want bool
	}{
		// Known source extensions.
		{".ts", true},
		{".tsx", true},
		{".go", true},
		{".py", true},
		{".rb", true},
		{".rs", true},
		{".java", true},
		{".js", true},
		{".jsx", true},
		{".c", true},
		{".cpp", true},
		{".h", true},
		{".hpp", true},
		{".cs", true},
		{".php", true},
		{".sh", true},
		{".sql", true},
		{".md", true},
		{".feature", true},
		{".FEATURE", true},
		{".yaml", true},
		{".yml", true},
		{".json", true},

		// Case-insensitive.
		{".TS", true},
		{".Go", true},
		{".JSON", true},

		// Without leading dot.
		{"go", true},
		{"md", true},

		// Non-source / unknown.
		{".com", false},
		{".html", false},
		{".css", false},
		{".txt", false},
		{".pdf", false},
		{".png", false},
		{"", false},
		{".", false},
	}
	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			t.Parallel()
			if got := IsSourceExtension(tt.ext); got != tt.want {
				t.Errorf("IsSourceExtension(%q) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}
