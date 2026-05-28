package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	types "github.com/npmanos/discourse-labeler/internal/pipeline"
)

type OzoneClient struct {
	Endpoint   string
	AdminToken string
	LabelerDID string
	HTTPClient *http.Client
}

func NewOzoneClient(endpoint, adminToken, labelerDID string) *OzoneClient {
	return &OzoneClient{
		Endpoint:   endpoint,
		AdminToken: adminToken,
		LabelerDID: labelerDID,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type queryLabelsResponse struct {
	Labels []struct {
		Val string `json:"val"`
		Src string `json:"src"`
	} `json:"labels"`
}

// IsAlreadyLabeled queries Ozone/Public AppView to check if we've already labeled this subject
func (oc *OzoneClient) IsAlreadyLabeled(ctx context.Context, targetURI string) (bool, error) {
	u, err := url.Parse(fmt.Sprintf("%s/xrpc/com.atproto.label.queryLabels", oc.Endpoint))
	if err != nil {
		return false, err
	}

	q := u.Query()
	q.Set("uriPatterns", targetURI)
	q.Set("sources", oc.LabelerDID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return false, err
	}

	resp, err := oc.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("queryLabels return non-200: %s", resp.Status)
	}

	var res queryLabelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, err
	}

	for _, lbl := range res.Labels {
		if lbl.Src == oc.LabelerDID && (lbl.Val == "meta-discourse" || lbl.Val == "possible-meta-discourse") {
			return true, nil
		}
	}

	return false, nil
}

// EmitLabel pushes an auto-moderation event adding the label to Ozone
func (oc *OzoneClient) EmitLabel(ctx context.Context, result *types.ClassificationResult) error {
	if result == nil || result.Post == nil {
		return fmt.Errorf("invalid classification result or missing post context")
	}
	labelVal := "possible-meta-discourse"
	if result.TargetPost.Classification == types.LabelDefiniteMeta {
		labelVal = "meta-discourse"
	}

	commentVal := formatOzoneComment(result)

	payload := map[string]interface{}{
		"event": map[string]interface{}{
			"$type":           "tools.ozone.moderation.defs#modEventLabel",
			"createLabelVals": []string{labelVal},
			"negateLabelVals": []string{},
			"comment":         commentVal,
		},
		"subject": map[string]interface{}{
			"$type": "com.atproto.repo.strongRef",
			"uri":   result.Post.TargetURI,
			"cid":   "", // Omit for general StrongRef parsing in Ozone
		},
		"createdBy": oc.LabelerDID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/xrpc/tools.ozone.moderation.emitEvent", oc.Endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oc.AdminToken))

	resp, err := oc.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("emitEvent non-success: %s", resp.Status)
	}

	return nil
}

func (oc *OzoneClient) EmitEscalation(ctx context.Context, result *types.ClassificationResult) error {
	if result == nil || result.Post == nil {
		return fmt.Errorf("invalid classification result or missing post context")
	}
	commentVal := formatOzoneComment(result)

	payload := map[string]interface{}{
		"event": map[string]interface{}{
			"$type":      "tools.ozone.moderation.defs#modEventEscalate",
			"comment":    commentVal,
			"escalateTo": "admin",
		},
		"subject": map[string]interface{}{
			"$type": "com.atproto.repo.strongRef",
			"uri":   result.Post.TargetURI,
			"cid":   "", // Omit for general StrongRef parsing in Ozone
		},
		"createdBy": oc.LabelerDID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/xrpc/tools.ozone.moderation.emitEvent", oc.Endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oc.AdminToken))

	resp, err := oc.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("emitEvent non-success: %s", resp.Status)
	}

	return nil
}

func formatOzoneComment(result *types.ClassificationResult) string {
	var sb strings.Builder
	sb.WriteString("[Target Post Analysis]\n")
	sb.WriteString(fmt.Sprintf("Classification: %s\n", result.TargetPost.Classification))
	sb.WriteString(fmt.Sprintf("Reasoning: %s\n", result.TargetPost.Reasoning))

	if result.ContextAnalysis.ParentPost != nil {
		sb.WriteString("\n[Context Analysis - Parent Post]\n")
		sb.WriteString(fmt.Sprintf("Classification: %s\n", result.ContextAnalysis.ParentPost.Classification))
		sb.WriteString(fmt.Sprintf("Reasoning: %s\n", result.ContextAnalysis.ParentPost.Reasoning))
	}

	if result.ContextAnalysis.QuotePost != nil {
		sb.WriteString("\n[Context Analysis - Quoted Post]\n")
		sb.WriteString(fmt.Sprintf("Classification: %s\n", result.ContextAnalysis.QuotePost.Classification))
		sb.WriteString(fmt.Sprintf("Reasoning: %s\n", result.ContextAnalysis.QuotePost.Reasoning))
	}

	return sb.String()
}
