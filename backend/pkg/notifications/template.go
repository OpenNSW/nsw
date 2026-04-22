package notifications

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sync"
	"text/template"
)

type templateCache struct {
	emailRoot string
	smsRoot   string

	mu           sync.RWMutex
	emailParsed  map[string]*template.Template
	smsParsed    map[string]*template.Template
}

func newTemplateCache(emailRoot, smsRoot string) *templateCache {
	return &templateCache{
		emailRoot:   emailRoot,
		smsRoot:     smsRoot,
		emailParsed: make(map[string]*template.Template),
		smsParsed:   make(map[string]*template.Template),
	}
}

// renderEmail executes the named email template and returns subject, plain body, and optional HTML body.
func (c *templateCache) renderEmail(id string, data map[string]any) (subject, plainBody, htmlBody string, err error) {
	tmpl, err := c.loadEmail(id)
	if err != nil {
		return "", "", "", err
	}

	var subjectBuf, plainBuf, htmlBuf bytes.Buffer
	if err = tmpl.ExecuteTemplate(&subjectBuf, "subject", data); err != nil {
		return "", "", "", fmt.Errorf("execute subject: %w", err)
	}
	if err = tmpl.ExecuteTemplate(&plainBuf, "plainBody", data); err != nil {
		return "", "", "", fmt.Errorf("execute plainBody: %w", err)
	}
	if tmpl.Lookup("htmlBody") != nil {
		if err = tmpl.ExecuteTemplate(&htmlBuf, "htmlBody", data); err != nil {
			return "", "", "", fmt.Errorf("execute htmlBody: %w", err)
		}
	}
	return subjectBuf.String(), plainBuf.String(), htmlBuf.String(), nil
}

// renderSMS executes the named SMS template and returns the rendered body.
func (c *templateCache) renderSMS(id string, data map[string]any) (string, error) {
	tmpl, err := c.loadSMS(id)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute SMS template: %w", err)
	}
	return buf.String(), nil
}

func (c *templateCache) loadEmail(id string) (*template.Template, error) {
	c.mu.RLock()
	tmpl, ok := c.emailParsed[id]
	c.mu.RUnlock()
	if ok {
		return tmpl, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if tmpl, ok = c.emailParsed[id]; ok {
		return tmpl, nil
	}

	path := filepath.Join(c.emailRoot, id+".tmpl")
	tmpl, err := template.New("").ParseFiles(path)
	if err != nil {
		return nil, fmt.Errorf("load email template %q: %w", id, err)
	}
	c.emailParsed[id] = tmpl
	return tmpl, nil
}

func (c *templateCache) loadSMS(id string) (*template.Template, error) {
	c.mu.RLock()
	tmpl, ok := c.smsParsed[id]
	c.mu.RUnlock()
	if ok {
		return tmpl, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if tmpl, ok = c.smsParsed[id]; ok {
		return tmpl, nil
	}

	path := filepath.Join(c.smsRoot, id+".txt")
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return nil, fmt.Errorf("load SMS template %q: %w", id, err)
	}
	c.smsParsed[id] = tmpl
	return tmpl, nil
}
