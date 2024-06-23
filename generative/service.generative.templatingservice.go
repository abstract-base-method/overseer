package generative

import (
	"overseer/common"
	"path"
	"text/template"
)

type defaultTemplatingService struct {
	spriteInternalTemplate *template.Template
	spriteExternalTemplate *template.Template
	coordinateTemplate     *template.Template
}

func NewTemplatingService() (TemplatingService, error) {
	tmpl, err := readTemplateFile(path.Join(common.GetConfiguration().Templating.TemplateBasePath, "map", "ollama", "spriteLore.internal.tmpl"))
	if err != nil {
		return nil, err
	}
	spriteInternalTmpl, err := template.New("spriteInternalGeneration").Parse(tmpl)
	if err != nil {
		return nil, err
	}

	tmpl, err = readTemplateFile(path.Join(common.GetConfiguration().Templating.TemplateBasePath, "map", "ollama", "spriteLore.external.tmpl"))
	if err != nil {
		return nil, err
	}
	spriteExternalTmpl, err := template.New("spriteExternalGeneration").Parse(tmpl)
	if err != nil {
		return nil, err
	}

	tmpl, err = readTemplateFile(path.Join(common.GetConfiguration().Templating.TemplateBasePath, "map", "ollama", "coordinateLore.tmpl"))
	if err != nil {
		return nil, err
	}
	coordinateLoreTmpl, err := template.New("coordinateLoreGeneration").Parse(tmpl)
	if err != nil {
		return nil, err
	}

	return &defaultTemplatingService{
		spriteInternalTemplate: spriteInternalTmpl,
		spriteExternalTemplate: spriteExternalTmpl,
		coordinateTemplate:     coordinateLoreTmpl,
	}, nil
}

func (s *defaultTemplatingService) InternalLoreTemplate() *template.Template {
	return s.spriteInternalTemplate
}

func (s *defaultTemplatingService) PublicLoreTemplate() *template.Template {
	return s.spriteExternalTemplate
}

func (s *defaultTemplatingService) CoordinateLoreTemplate() *template.Template {
	return s.coordinateTemplate
}
