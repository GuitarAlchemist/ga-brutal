package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Compile and execute the production budget statements, without loading BRUTAL,
// opening a window or reimplementing the formula in the test's language.
func TestTrafficCapacity(t *testing.T) {
	if os.Getenv("SNAPSHOT_DOTNET_TESTS") != "1" {
		t.Skip("enable after installing the pinned .NET SDK")
	}
	source, err := os.ReadFile(filepath.Join(repo, "Experiments/Brutal/Colossus/Program.cs"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "        // Every instance shares one buffer:")
	if start < 0 {
		t.Fatal("budget seam missing")
	}
	end := strings.Index(text[start:], "        if (swarm > swarmRoom)")
	if end < 0 {
		t.Fatal("budget end missing")
	}
	budget := text[start : start+end]
	fixture := `using System;
using System.Linq;
static int Capacity(int statics, int traffic) {
 const int MaxSwarm = 40000, MaxInstances = MaxSwarm + 1024;
 var staticInstances = new int[statics];
 var _lanes = new[] { new { Count = traffic } };
` + budget + `
 return swarmRoom;
}
foreach (var c in new[] { (0,0,40000), (1024,0,40000), (2000,281,38743), (40743,281,0) }) {
 var actual = Capacity(c.Item1,c.Item2);
 if (actual != c.Item3) throw new Exception($"{c}: got {actual}");
 if (c.Item1 + actual + c.Item2 > 41024) throw new Exception("instance capacity exceeded");
}
Console.WriteLine("Traffic capacity: four boundary cases pass");
`
	root := t.TempDir()
	project := `<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><OutputType>Exe</OutputType><TargetFramework>net10.0</TargetFramework></PropertyGroup></Project>`
	if err := os.WriteFile(filepath.Join(root, "Test.csproj"), []byte(project), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Program.cs"), []byte(fixture), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("dotnet", "run", "--project", filepath.Join(root, "Test.csproj"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("production budget: %s\n%v", out, err)
	}
}
