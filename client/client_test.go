package client

import (
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/ecs"
)

func TestMergeEnv(t *testing.T) {

	c := Client{
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}

	env := make(map[string]string)

	def := &ecs.ContainerDefinition{
		Environment: make([]*ecs.KeyValuePair, 0),
	}

	c.merge(def, &env)

	if len(def.Environment) != 0 {
		t.Error("Expected 0 environment variables")
	}
}

func TestMergeSingleVar(t *testing.T) {

	c := Client{
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}

	env := make(map[string]string, 1)
	env["aa"] = "1"

	def := &ecs.ContainerDefinition{
		Environment: make([]*ecs.KeyValuePair, 0),
	}

	c.merge(def, &env)

	if len(def.Environment) != 1 {
		t.Errorf("Expected 1 environment variables got %d ", len(def.Environment))
	}

	if !contains(def.Environment, "aa", "1") {
		t.Error("expected aa:1")
	}
}

func TestMergeAddTwoVars(t *testing.T) {

	c := Client{
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}

	env := make(map[string]string, 1)
	env["aa"] = "1"
	env["ab"] = "2"

	def := &ecs.ContainerDefinition{
		Environment: make([]*ecs.KeyValuePair, 0),
	}

	c.merge(def, &env)

	if len(def.Environment) != 2 {
		t.Errorf("Expected 2 environment variables got %d ", len(def.Environment))
	}

	if !contains(def.Environment, "aa", "1") {
		t.Error("expected aa:1")
	}

	if !contains(def.Environment, "ab", "2") {
		t.Error("expected ab:2")
	}
}

func TestMergeTwoVars(t *testing.T) {

	c := Client{
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}

	env := make(map[string]string, 1)
	env["aa"] = "1"
	env["ab"] = "2"

	def := &ecs.ContainerDefinition{
		Environment: make([]*ecs.KeyValuePair, 0),
	}
	cvar := ecs.KeyValuePair{}
	cvar.SetName("ab")
	cvar.SetValue("*")
	def.Environment = append(def.Environment, &cvar)

	c.merge(def, &env)

	if len(def.Environment) != 2 {
		t.Errorf("Expected 2 environment variables got %d ", len(def.Environment))
	}

	if !contains(def.Environment, "aa", "1") {
		t.Error("expected aa:1")
	}

	if !contains(def.Environment, "ab", "2") {
		t.Error("expected ab:2")
	}
}

func contains(env []*ecs.KeyValuePair, name, value string) bool {
	for _, v := range env {
		if *v.Name == name && *v.Value == value {
			return true
		}
	}
	return false
}
