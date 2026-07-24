package opencloud

import (
	"bytes"
	"fmt"
)

// UploadAnimation uploads an animation using Open Cloud API
func UploadAnimation(openCloudClient *Client, name string, description string, data *bytes.Buffer) (int64, error) {
	if openCloudClient == nil {
		return 0, fmt.Errorf("open cloud client is nil")
	}

	if data == nil || data.Len() == 0 {
		return 0, fmt.Errorf("animation data is empty")
	}

	// Copy buffer to avoid consuming the original
	buf := make([]byte, data.Len())
	copy(buf, data.Bytes())

	resp, err := openCloudClient.UploadAsset("Animation", buf, name, description)
	if err != nil {
		return 0, err
	}

	return resp.AssetID, nil
}

// UploadMesh uploads a mesh using Open Cloud API
func UploadMesh(openCloudClient *Client, name string, description string, data *bytes.Buffer) (int64, error) {
	if openCloudClient == nil {
		return 0, fmt.Errorf("open cloud client is nil")
	}

	if data == nil || data.Len() == 0 {
		return 0, fmt.Errorf("mesh data is empty")
	}

	buf := make([]byte, data.Len())
	copy(buf, data.Bytes())

	resp, err := openCloudClient.UploadAsset("Mesh", buf, name, description)
	if err != nil {
		return 0, err
	}

	return resp.AssetID, nil
}
