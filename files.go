package fluxer

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"strconv"
)

type File struct {
	Name        string
	Description string
	Data        []byte
}

func (c *Client) DoFiles(ctx context.Context, method, path string, body any, files []File, out any) error {
	if len(files) == 0 {
		return c.Do(ctx, method, path, body, out)
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	payload, err := json.Marshal(withAttachments(body, files))
	if err != nil {
		return err
	}
	if err := w.WriteField("payload_json", string(payload)); err != nil {
		return err
	}
	for i, f := range files {
		part, err := w.CreateFormFile("files["+strconv.Itoa(i)+"]", f.Name)
		if err != nil {
			return err
		}
		if _, err := part.Write(f.Data); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.send(ctx, method, path, w.FormDataContentType(), buf.Bytes(), body, out)
}

type attachmentMeta struct {
	ID          int    `json:"id"`
	Filename    string `json:"filename"`
	Description string `json:"description,omitempty"`
}

func withAttachments(body any, files []File) any {
	described := false
	for _, f := range files {
		described = described || f.Description != ""
	}
	if !described {
		return body
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return body
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return body
	}
	meta := make([]attachmentMeta, len(files))
	for i, f := range files {
		meta[i] = attachmentMeta{ID: i, Filename: f.Name, Description: f.Description}
	}
	fields["attachments"], _ = json.Marshal(meta)
	return fields
}
