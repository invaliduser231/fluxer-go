package fluxer

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"strconv"
)

// File is a file sent with DoFiles or Upload. Name is the file name and
// Description is sent as the attachment description.
type File struct {
	Name        string
	Description string
	Data        []byte
}

// DoFiles sends a multipart/form-data request with body encoded as the
// payload_json field and files as files[0], files[1] and so on, and decodes the
// response into out. When any file has a Description, an attachments array with
// ids, filenames and descriptions is added to the payload; this only works when
// body encodes as a JSON object, otherwise the descriptions are dropped. Without
// files DoFiles behaves like Do. Retries follow the same rules as Do.
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
