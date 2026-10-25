package fluxer

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"slices"
	"strconv"
)

// File is a file sent with DoFiles or Upload. Name is the file name. Title,
// Description and Flags, for example AttachmentSpoiler, are sent with the
// attachment. Duration in seconds and the base64 Waveform describe a voice
// message.
type File struct {
	Name        string
	Title       string
	Description string
	Flags       int
	Duration    int
	Waveform    string
	Data        []byte
}

func (f File) hasMeta() bool {
	return f.Title != "" || f.Description != "" || f.Flags != 0 || f.Duration != 0 || f.Waveform != ""
}

// DoFiles sends a multipart/form-data request with body encoded as the
// payload_json field and files as files[0], files[1] and so on, and decodes the
// response into out. When any file has a title, description, flags, duration or
// waveform, an attachments array with ids, filenames and those fields is added
// to the payload; this only works when body encodes as a JSON object, otherwise
// they are dropped. Without
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
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Flags       int    `json:"flags,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	Waveform    string `json:"waveform,omitempty"`
}

func withAttachments(body any, files []File) any {
	if !slices.ContainsFunc(files, File.hasMeta) {
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
		meta[i] = attachmentMeta{ID: i, Filename: f.Name, Title: f.Title, Description: f.Description, Flags: f.Flags, Duration: f.Duration, Waveform: f.Waveform}
	}
	fields["attachments"], _ = json.Marshal(meta)
	return fields
}
