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

var dropFlag = flag.Bool("drop", false, "drops replace directives specifies in configuration file")

type Config struct {
	Targets []string            `yaml:"targets"`
	Replace []ReplaceDirectives `yaml:"replace"`

	abs map[string]struct{} // filepath.Abs of Targets
}

func (c *Config) HasTarget(abspath string) bool {
	_, ok := c.abs[abspath]
	return ok
}

type ReplaceDirectives struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

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
	if err := scanFiles(cwd, cfg); err != nil {
		fatal(err)
	}
}

func scanFiles(path string, cfg Config) error {
	des, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, de := range des {
		name := de.Name()
		fullpath := filepath.Join(path, name)
		if filepath.Base(name) == "go.mod" {
			if cfg.HasTarget(filepath.Dir(fullpath)) {
				inf, err := de.Info()
				if err != nil {
					return err
				}
				if err := patchmod(fullpath, inf.Mode(), cfg); err != nil {
					return err
				}
			}
		}
		if de.IsDir() {
			if err := scanFiles(fullpath, cfg); err != nil {
				return err
			}
		}
	}
	return nil
}

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
