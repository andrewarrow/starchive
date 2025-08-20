package handlers

import (
	"fmt"
	"os"
	"os/exec"

	"starchive/blend"
	"starchive/util"
)

func HandleBlend() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: starchive blend <id1> <id2>")
		fmt.Println("Example: starchive blend OIduTH7NYA8 EbD7lfrsY2s")
		fmt.Println("Enters an interactive blend shell with real-time controls.")
		os.Exit(1)
	}

	id1 := os.Args[2]
	id2 := os.Args[3]
	
	// Initialize database
	db, err := util.InitDatabase()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	
	blendShell := blend.NewShell(id1, id2, db)
	blendShell.Run()
}

func HandleBlendClear() {
	if len(os.Args) == 2 {
		clearAllBlendMetadata()
	} else if len(os.Args) == 4 {
		id1 := os.Args[2]
		id2 := os.Args[3]
		clearSpecificBlendMetadata(id1, id2)
	} else {
		fmt.Println("Usage: starchive blend-clear [id1 id2]")
		fmt.Println("  starchive blend-clear          Clear all blend metadata")
		fmt.Println("  starchive blend-clear id1 id2  Clear metadata for specific track pair")
		os.Exit(1)
	}
}

func clearAllBlendMetadata() {
	tmpDir := "/tmp"
	pattern := "starchive_blend_*.tmp"
	
	cmd := exec.Command("find", tmpDir, "-name", pattern, "-type", "f", "-delete")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error clearing blend metadata: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("All blend metadata cleared.")
}

func clearSpecificBlendMetadata(id1, id2 string) {
	file1 := "/tmp/starchive_blend_" + id1 + "_" + id2 + ".tmp"
	file2 := "/tmp/starchive_blend_" + id2 + "_" + id1 + ".tmp"
	
	removed := false
	
	if _, err := os.Stat(file1); err == nil {
		os.Remove(file1)
		removed = true
	}
	
	if _, err := os.Stat(file2); err == nil {
		os.Remove(file2)
		removed = true
	}
	
	if removed {
		fmt.Printf("Blend metadata cleared for tracks %s and %s.\n", id1, id2)
	} else {
		fmt.Printf("No blend metadata found for tracks %s and %s.\n", id1, id2)
	}
}