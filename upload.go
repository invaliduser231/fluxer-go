package fluxer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
)

const maxUploadsPerPlan = 10

// UploadPart is one part of a multipart upload and the presigned URL it is
// sent to. PartNumber starts at 1.
type UploadPart struct {
	PartNumber int    `json:"part_number"`
	UploadURL  string `json:"upload_url"`
}

// UploadPlan is the server's upload instruction for one file. UploadMode is
// "singlepart", where the whole file goes to UploadURL, or "multipart", where
// the file is split into Parts of PartSize bytes.
type UploadPlan struct {
	ID             int          `json:"id"`
	Filename       string       `json:"filename"`
	FileSize       int64        `json:"file_size"`
	ContentType    string       `json:"content_type"`
	UploadFilename string       `json:"upload_filename"`
	UploadMode     string       `json:"upload_mode"`
	UploadURL      string       `json:"upload_url,omitempty"`
	UploadID       string       `json:"upload_id,omitempty"`
	PartSize       int64        `json:"part_size,omitempty"`
	Parts          []UploadPart `json:"parts,omitempty"`
}

// UploadedFile describes a file uploaded with Upload. Add it to
// MessageSend.Attachments to attach the file to a message.
type UploadedFile struct {
	ID             int    `json:"id"`
	Filename       string `json:"filename"`
	ContentType    string `json:"content_type"`
	UploadFilename string `json:"upload_filename"`
	FileSize       int64  `json:"file_size"`
	Description    string `json:"description,omitempty"`
}

func contentType(f File) string {
	if t := mime.TypeByExtension(filepath.Ext(f.Name)); t != "" {
		return t
	}
	return http.DetectContentType(f.Data)
}

// Upload uploads files to storage through presigned URLs and returns them in
// a form that can be attached to a message in channelID. It requests an upload
// plan with POST /channels/{id}/attachments, PUTs each file or part, and
// finishes multipart uploads with POST /channels/{id}/attachments/complete. At
// most 10 files are accepted per call. The content type is guessed from the file
// extension, falling back to the data.
//
// The PUT requests use the transport of the Client's HTTP client but neither its
// timeout nor the Authorization header, so bound them with ctx.
func (c *Client) Upload(ctx context.Context, channelID string, files []File) ([]UploadedFile, error) {
	if len(files) == 0 {
		return nil, nil
	}
	if len(files) > maxUploadsPerPlan {
		return nil, fmt.Errorf("fluxer: at most %d files can be uploaded at once", maxUploadsPerPlan)
	}
	type item struct {
		ID          int    `json:"id"`
		Filename    string `json:"filename"`
		FileSize    int64  `json:"file_size"`
		ContentType string `json:"content_type"`
	}
	req := struct {
		Attachments []item `json:"attachments"`
	}{}
	for i, f := range files {
		req.Attachments = append(req.Attachments, item{ID: i, Filename: f.Name, FileSize: int64(len(f.Data)), ContentType: contentType(f)})
	}
	var plan struct {
		Attachments []UploadPlan `json:"attachments"`
	}
	if err := c.Do(ctx, http.MethodPost, channelPath(channelID)+"/attachments", req, &plan); err != nil {
		return nil, err
	}
	type pending struct {
		UploadFilename string `json:"upload_filename"`
		UploadID       string `json:"upload_id"`
	}
	var multipart []pending
	out := make([]UploadedFile, 0, len(plan.Attachments))
	for _, p := range plan.Attachments {
		if p.ID < 0 || p.ID >= len(files) {
			return nil, fmt.Errorf("fluxer: upload plan names unknown file %d", p.ID)
		}
		f := files[p.ID]
		switch p.UploadMode {
		case "singlepart":
			if err := c.put(ctx, p.UploadURL, p.ContentType, f.Data); err != nil {
				return nil, err
			}
		case "multipart":
			if p.PartSize <= 0 {
				return nil, errors.New("fluxer: upload plan has no part size")
			}
			for _, part := range p.Parts {
				start := int64(part.PartNumber-1) * p.PartSize
				if start < 0 || start >= int64(len(f.Data)) {
					return nil, fmt.Errorf("fluxer: upload plan part %d is outside the file", part.PartNumber)
				}
				end := min(start+p.PartSize, int64(len(f.Data)))
				if err := c.put(ctx, part.UploadURL, "", f.Data[start:end]); err != nil {
					return nil, err
				}
			}
			multipart = append(multipart, pending{UploadFilename: p.UploadFilename, UploadID: p.UploadID})
		default:
			return nil, fmt.Errorf("fluxer: unknown upload mode %q", p.UploadMode)
		}
		out = append(out, UploadedFile{
			ID: p.ID, Filename: p.Filename, ContentType: p.ContentType, UploadFilename: p.UploadFilename,
			FileSize: p.FileSize, Description: f.Description,
		})
	}
	if len(out) != len(files) {
		return nil, fmt.Errorf("fluxer: upload plan covers %d of %d files", len(out), len(files))
	}
	if len(multipart) > 0 {
		body := map[string][]pending{"uploads": multipart}
		if err := c.Do(ctx, http.MethodPost, channelPath(channelID)+"/attachments/complete", body, nil); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *Client) put(ctx context.Context, target, contentType string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(data))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	client := &http.Client{Transport: c.httpClient().Transport}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("fluxer: upload failed with %d: %s", resp.StatusCode, bytes.TrimSpace(msg))
	}
	return nil
}

// SendMessageUploads uploads files with Upload and sends m to channelID with
// the uploaded files appended to its attachments.
func (c *Client) SendMessageUploads(ctx context.Context, channelID string, m MessageSend, files []File) (*Message, error) {
	uploaded, err := c.Upload(ctx, channelID, files)
	if err != nil {
		return nil, err
	}
	m.Attachments = append(m.Attachments, uploaded...)
	return c.SendMessage(ctx, channelID, m)
}

// RefreshAttachmentURLs requests new signed URLs for the given attachment URLs
// with POST /attachments/refresh-urls. The result maps each original URL to its
// new URL; URLs the API did not refresh are missing from the map.
func (c *Client) RefreshAttachmentURLs(ctx context.Context, urls []string) (map[string]string, error) {
	var r struct {
		RefreshedURLs []struct {
			Original  string `json:"original"`
			Refreshed string `json:"refreshed"`
		} `json:"refreshed_urls"`
	}
	if err := c.Do(ctx, http.MethodPost, "/attachments/refresh-urls", map[string][]string{"attachment_urls": urls}, &r); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(r.RefreshedURLs))
	for _, u := range r.RefreshedURLs {
		out[u.Original] = u.Refreshed
	}
	return out, nil
}
