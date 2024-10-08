package test

import (
	"testing"
	"io"
	"os"
	"path/filepath"
	"github.com/KevinJGard/MusicDB/src/model"
	"github.com/dhowden/tag"
	"github.com/stretchr/testify/assert"
)

func copyMp3(tempDir string, fileName string) error {
	file, err := os.Open(filepath.Join("..", "..", "testdata", "testdata_with_tags_sample.id3v24.mp3"))
    if err != nil {
        return err
    }
    defer file.Close()

    destFile, err := os.Create(filepath.Join(tempDir, fileName))
    if err != nil {
        return err
    }
    defer destFile.Close()

    _, err = io.Copy(destFile, file)
    return err
}

func createTempDirectoryWithFiles(tempDir string, files []string) error {
    for _, file := range files {
        if err := copyMp3(tempDir, file); err != nil {
	    	return err
	    }
    }
    return nil
}

func TestFindMP3Files(t *testing.T) {
	files := []string{"test1.mp3", "test2.mp3", "test3.mp3", "test4.txt"}
	tempDir := t.TempDir()
	err := createTempDirectoryWithFiles(tempDir, files)
	assert.NoError(t, err, "Failed to create temp dir with files.")

	miner := model.NewMiner()
	foundFiles, err := miner.FindMP3Files(tempDir)
	assert.NoError(t, err, "Error traversing directory %s.", tempDir)
	
	assert.Len(t, foundFiles, 3, "Expected %d MP3 files, but found %d.", 3, len(foundFiles))

	for _, file := range files[:3] {
		filePath := filepath.Join(tempDir, file)
		assert.Contains(t, foundFiles, filePath, "Expected file %s not found in results.", filePath)
	}
}

func TestMineMetadata(t *testing.T) {
	files := []string{"test1.mp3", "test2.mp3", "test3.mp3"}
	tempDir := t.TempDir()
	err := createTempDirectoryWithFiles(tempDir, files)
	assert.NoError(t, err, "Failed to create temp dir with files.")

	miner := model.NewMiner()

	for _, file := range files {
		filePath := filepath.Join(tempDir, file)
		metadata, err := miner.MineMetadata(filePath)
		assert.NoError(t, err, "Error reading metadata for %s.", file)
		
		assert.NotEmpty(t, metadata["Title"], "Tag \"Title\" not found in %s.", filePath)
		assert.NotEmpty(t, metadata["Artist"], "Tag \"Artist\" not found in %s.", filePath)
		assert.NotEmpty(t, metadata["Album"], "Tag \"Album\" not found in %s.", filePath)
		assert.NotEmpty(t, metadata["Genre"], "Tag \"Genre\" not found in %s.", filePath)
		assert.NotZero(t, metadata["Year"], "Tag \"Year\" not found in %s.", filePath)
		track := metadata["Track"].(map[string]int)
		assert.NotZero(t, track["Number"], "Tag \"Track\" number not found in %s.", filePath)
		assert.NotZero(t, track["Total"], "Tag \"Track\" total not found in %s.", filePath)
	}
}

func TestAssignTag(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "..", "testdata", "testdata_without_tags_sample.mp3"))
    assert.NoError(t, err, "Error opening test file.")
    defer file.Close()

    metadata, err := tag.ReadFrom(file)
    assert.NoError(t, err, "Error reading metadata.")

	miner := model.NewMiner()
	tags := miner.AssignTag(metadata)
	assert.NotEmpty(t, tags["Title"], "Expected a valid Title or \"Unknown\".")
	assert.NotEmpty(t, tags["Artist"], "Expected a valid Artist or \"Unknown\".")
	assert.NotEmpty(t, tags["Album"], "Expected a valid Album or \"Unknown\".")
	assert.NotEmpty(t, tags["Title"], "Expected a valid Title or \"Unknown\".")
	assert.NotZero(t, tags["Year"], "Expected a valid Year or \"1\".")
	track := tags["Track"].(map[string]int)
	assert.NotZero(t, track["Number"], "Expected valid Track number or \"1\".")
	assert.NotZero(t, track["Total"], "Expected valid Track total or \"1\".")
}

func TestProcessFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := "test.mp3"

	err := copyMp3(tempDir, testFile)
	assert.NoError(t, err, "Failed to copy test file.")

	db := model.NewDataBase()
	miner := model.NewMiner()
	filePath := filepath.Join(tempDir, testFile)
    err = miner.ProcessFile(db, filePath)
    assert.NoError(t, err, "Error processing the file.")

    var id int
	query := `SELECT id_rola FROM rolas WHERE title = ?`
	err = db.Db.QueryRow(query, "Test Title").Scan(&id)
	assert.NoError(t, err, "Expected no error while querying for inserted song.")
	assert.NotZero(t, id, "Expected song to be inserted into the database.")

	query = `SELECT id_performer FROM performers WHERE name = ?`
	err = db.Db.QueryRow(query, "Test Artist").Scan(&id)
	assert.NoError(t, err, "Expected no error while querying for inserted performer.")
	assert.NotZero(t, id, "Expected performer to be inserted into the database.")

	query = `SELECT id_album FROM albums WHERE name = ?`
	err = db.Db.QueryRow(query, "Test Album").Scan(&id)
	assert.NoError(t, err, "Expected no error while querying for inserted album.")
	assert.NotZero(t, id, "Expected album to be inserted into the database.")

	defer db.Db.Close()
}