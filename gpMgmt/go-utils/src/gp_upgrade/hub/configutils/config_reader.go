package configutils

import (
	"encoding/json"
	"sync"

	"gp_upgrade/utils"

	"github.com/pkg/errors"
)

type Reader struct {
	config       SegmentConfiguration
	fileLocation string
	mu           sync.RWMutex
}

func NewReader() Reader {
	return Reader{}
}

func (reader *Reader) OfOldClusterConfig(base string) {
	reader.fileLocation = GetConfigFilePath(base)
	reader.config = nil
}

func (reader *Reader) OfNewClusterConfig(base string) {
	reader.fileLocation = GetNewClusterConfigFilePath(base)
	reader.config = nil
}

func (reader *Reader) Read() error {
	reader.mu.RLock()
	defer reader.mu.RUnlock()

	if reader.fileLocation == "" {
		return errors.New("Reader file location unknown")
	}

	contents, err := utils.System.ReadFile(reader.fileLocation)

	if err != nil {
		return errors.New(err.Error())
	}
	err = json.Unmarshal([]byte(contents), &reader.config)
	if err != nil {
		return errors.New(err.Error())
	}
	return nil
}

// returns -1 for not found
func (reader *Reader) GetPortForSegment(segmentDbid int) int {
	reader.mu.RLock()
	defer reader.mu.RUnlock()

	result := -1
	if len(reader.config) == 0 {
		err := reader.Read()
		if err != nil {
			return result
		}
	}

	for i := 0; i < len(reader.config); i++ {
		segment := reader.config[i]
		if segment.Dbid == segmentDbid {
			result = segment.Port
			break
		}
	}

	return result
}

func (reader *Reader) GetSegmentHostnames() ([]string, error) {
	return reader.getHostnames(false, -1)
}

func (reader *Reader) GetHostnames() ([]string, error) {
	return reader.getHostnames(false, -2)
}

// if matchContentID is true, return the hostname of the provided contentID
// else, return hostnames for all contentIDs not matching the provided contentID
// practically, false -1 return all hosts in the cluster except master
// false -2 returns all hosts in the cluster
func (reader *Reader) getHostnames(matchContentID bool, contentID int) ([]string, error) {
	reader.mu.RLock()
	defer reader.mu.RUnlock()

	if len(reader.config) == 0 {
		err := reader.Read()
		if err != nil {
			return nil, err
		}
	}

	hostnamesSeen := make(map[string]bool)
	for i := 0; i < len(reader.config); i++ {
		hostName := reader.config[i].Hostname
		_, contained := hostnamesSeen[hostName]
		if !contained {
			if matchContentID && contentID == reader.config[i].Content {
				hostnamesSeen[reader.config[i].Hostname] = true
			} else if !matchContentID && contentID != reader.config[i].Content {
				hostnamesSeen[reader.config[i].Hostname] = true
			}
		}
	}
	var hostnames []string
	for k := range hostnamesSeen {
		hostnames = append(hostnames, k)
	}
	return hostnames, nil
}

func (reader *Reader) GetSegmentConfiguration() SegmentConfiguration {
	reader.mu.RLock()
	defer reader.mu.RUnlock()

	if len(reader.config) == 0 {
		err := reader.Read()
		if err != nil {
			return nil
		}
	}

	return reader.config
}

func (reader *Reader) GetMasterDataDir() string {
	config := reader.GetSegmentConfiguration()
	for i := 0; i < len(config); i++ {
		segment := config[i]
		if segment.Content == -1 {
			return segment.Datadir
		}
	}
	return ""
}

func (reader *Reader) GetMaster() *Segment {
	var nilSegment *Segment
	config := reader.GetSegmentConfiguration()
	for i := 0; i < len(config); i++ {
		segment := config[i]
		if segment.Content == -1 {
			return &segment
		}
	}
	return nilSegment
}
