package condition

import (
	"encoding/json"
	"fmt"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/rule"
	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

func parseSearch(t tokens, data types.Detection, c rule.Config) (match.Branch, error) {
	fmt.Printf("Parsing %+v\n", t)

	// seek to LPAR -> store offset set balance as 1
	// seek from offset to end -> increment balance when encountering LPAR, decrement when encountering RPAR
	// increment group count on every decrement
	// stop when balance is 0, error of EOF if balance is positive or negative
	// if group count is > 0, fill sub brances via recursion
	// finally, build branch from identifiers and logic statements

	if t.contains(IdentifierAll) {
		return nil, fmt.Errorf("TODO - THEM identifier")
	}
	if t.contains(IdentifierWithWildcard) {
		return nil, fmt.Errorf("TODO - wildcard identifier")
	}
	if t.contains(StOne) || t.contains(StAll) {
		return nil, fmt.Errorf("TODO - X of statement")
	}

	// pass 1 - discover groups
	groups, ok, err := newGroupOffsetInTokens(t)
	if err != nil {
		return nil, err
	}
	if ok {
		j, _ := json.Marshal(groups)
		fmt.Printf("%s\n", data["condition"].(string))
		fmt.Printf("got %d groups offsets are %s\n", len(groups), string(j))
		return nil, fmt.Errorf("TODO - implement parsing sub-groups recursively")
	}

	return parseSimpleSearch(t, data, c)
}

// simple search == just a valid group sequence with no sub-groups
// maybe will stay, maybe exists just until I figure out the parse logic
func parseSimpleSearch(t tokens, data types.Detection, c rule.Config) (match.Branch, error) {
	var (
		negated   bool
		rules     = make([]match.Branch, 0)
		modifiers = []Token{TokNil}
	)
	for _, item := range t {
		switch item.T {
		case KeywordNot:
			negated = true
		case KeywordAnd:
			modifiers = append(modifiers, KeywordAnd)
		case KeywordOr:
			modifiers = append(modifiers, KeywordOr)
		case Identifier:
			r, err := newRuleMatcherFromIdent(data.Get(item.Val), c.LowerCase)
			if err != nil {
				return nil, err
			}
			// no modifier on this rule, mark it as such for second pass
			if len(modifiers)-1 != len(rules) {
				modifiers = append(modifiers, TokNil)
			}
			rules = append(rules, func() match.Branch {
				if negated {
					return match.NodeNot{Branch: r}
				}
				return r
			}())
			// reset modifiers
			negated = false
		}
	}

	return nil, fmt.Errorf("WIP")
}

type parser struct {
	lex *lexer

	// maintain a list of collected and validated tokens
	tokens

	// memorize last token to validate proper sequence
	// for example, two identifiers have to be joined via logical AND or OR, otherwise the sequence is invalid
	previous Token

	// sigma detection map that contains condition query and relevant fields
	sigma types.Detection

	// for debug
	condition string

	// sigma condition rules
	rules []interface{}
}

func (p *parser) run() error {
	if p.lex == nil {
		return fmt.Errorf("cannot run condition parser, lexer not initialized")
	}
	// Pass 1: collect tokens, do basic sequence validation and collect sigma fields
	if err := p.collectAndValidateTokenSequences(); err != nil {
		return err
	}
	// Pass 2: find groups
	fmt.Println("------------------")
	if _, err := parseSearch(p.tokens, p.sigma, rule.Config{}); err != nil {
		return err
	}
	return nil
}

func (p *parser) collectAndValidateTokenSequences() error {
	for item := range p.lex.items {

		if item.T == TokUnsupp {
			return types.ErrUnsupportedToken{Msg: item.Val}
		}
		if !validTokenSequence(p.previous, item.T) {
			return fmt.Errorf(
				"invalid token sequence %s -> %s. Value: %s",
				p.previous,
				item.T,
				item.Val,
			)
		}
		if item.T != LitEof {
			p.tokens = append(p.tokens, item)
		}
		p.previous = item.T
	}
	if p.previous != LitEof {
		return fmt.Errorf("last element should be EOF, got %s", p.previous.String())
	}
	return nil
}
