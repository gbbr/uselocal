package main

import (
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v2"
)

// dropFlag specifies that the replace directives should be dropped, instead of added.
var dropFlag = flag.Bool("drop", false, "drops the replace directives specified in the configuration file")

// Config specifies the configuration for uselocal.
type Config struct {
	// Targets specifies a list of target sub-folders where go.mod files
	// should be looked for.
	Targets []string `yaml:"targets"`

	// Replace specifies the set of replace directives to apply.
	Replace []ReplaceDirectives `yaml:"replace"`

	// abs specifies an easy lookup method for targets. Its keys are
	// the same as Targets, but as absolute paths.
	abs map[string]struct{}
}

// HasTarget reports whether the given absolute path exists in the configuration targets.
func (c *Config) HasTarget(abspath string) bool {
	_, ok := c.abs[abspath]
	return ok
}

// ReplaceDirectives specifies a replace directive.
type ReplaceDirectives struct {
	// From specifies the module that should be replaced.
	From string `yaml:"from"`
	// To specifies the path that it should be replaced with.
	To string `yaml:"to"`
}

// NewConfig creates and returns a new configuration from the YAML file based at path,
// along with any errors occurred.
func NewConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, err
	}
	cfg.abs = make(map[string]struct{})
	for _, t := range cfg.Targets {
		v, err := filepath.Abs(t)
		if err != nil {
			return cfg, err
		}
		cfg.abs[v] = struct{}{}
	}
	for i, rd := range cfg.Replace {
		// rewrite target paths because relative ones will not apply in subfolders
		v, err := filepath.Abs(rd.To)
		if err != nil {
			return cfg, err
		}
		cfg.Replace[i].To = v
	}
	return cfg, nil
}

func main() {
	flag.Parse()
	fatal := func(err error) { log.Fatal(err) }
	file := "./.uselocal.yaml"
	if v, ok := os.LookupEnv("USELOCAL"); ok {
		file = v
	}
	cfg, err := NewConfig(file)
	if err != nil {
		fatal(err)
	}
	cwd, err := filepath.Abs(".")
	if err != nil {
		fatal(err)
	}
	if err := rewriteModFiles(cwd, cfg); err != nil {
		fatal(err)
	}
}

// rewriteModFiles rewrites all go.mod files existing in path based on the given config.
func rewriteModFiles(path string, cfg Config) error {
	des, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, de := range des {
		name := de.Name()
		fullpath := filepath.Join(path, name)
		if de.IsDir() {
			// recursively progress into all subfolders
			if err := rewriteModFiles(fullpath, cfg); err != nil {
				return err
			}
		}
		if filepath.Base(name) != "go.mod" {
			// we only care about go.mod files
			continue
		}
		if !cfg.HasTarget(filepath.Dir(fullpath)) {
			// ...that are specified as targets
			continue
		}
		inf, err := de.Info()
		if err != nil {
			return err
		}
		if err := patchmod(fullpath, inf.Mode(), cfg); err != nil {
			return err
		}
	}
	return nil
}

// patchmod patches the go.mod file found at path, using the given config and file mode.
// Any error is returned.
func patchmod(path string, mode fs.FileMode, cfg Config) error {
	slurp, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	f, err := modfile.Parse(path, slurp, nil)
	if err != nil {
		return err
	}
	for _, rd := range cfg.Replace {
		if *dropFlag {
			if err := f.DropReplace(rd.From, ""); err != nil {
				return err
			}
		} else {
			if err := f.AddReplace(rd.From, "", rd.To, ""); err != nil {
				return err
			}
		}
	}
	f.Cleanup()
	out, err := f.Format()
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, mode)
}
