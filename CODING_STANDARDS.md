# Coding Standards

This project follows standard Go conventions and `gofmt`. Keep the code idiomatic and avoid unnecessary stylistic complexity.

## Blank Lines

Use blank lines to separate logical steps or groups of operations within a function. Do not avoid blank lines merely because a function is short.

In particular:

- Group related guard clauses and early returns when they form a coherent sequence.
- Add a blank line when transitioning from validation/guard clauses to the main operation.
- Add a blank line before a return statement when it improves readability; this is usually appropriate when the return represents the function's final result.
- Use blank lines inside loops when separating validation or checks from the operation that follows.
- Do not add blank lines between every individual statement or between closely related if statements.

For example:

```Go
func doSomething(cfg config.Config, data []byte) ([]byte, error) {
	if len(cfg.Data) == 0 {
		return nil, nil
	}
	if err := validate(cfg.Data); err != nil {
		return nil, err
	}

	parsed, err := parseCatalog(data)
	if err != nil {
		return nil, err
	}

	return render(cfg, parsed)
}
```

Prefer grouping related conditions and operations rather than packing an entire function into one visually continuous block:

```Go
func validate(entries []config.Entry) error {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Name == "" {
			return fmt.Errorf("entry name is empty")
		}
		if _, ok := seen[entry.Name]; ok {
			return fmt.Errorf("duplicate entry name %q", entry.Name)
		}

		seen[entry.Name] = struct{}{}
		if len(entry.Matchers) == 0 {
			return fmt.Errorf("entry %q has no matchers", entry.Name)
		}

		for _, matcher := range entry.Matchers {
			if matcher.Type == "" {
				return fmt.Errorf("entry %q has a matcher with empty type", entry.Name)
			}
			if matcher.Name == "" {
				return fmt.Errorf("entry %q has a matcher with empty name", entry.Name)
			}
			if !knownType(matcher.Type) {
				return fmt.Errorf("unknown type %q", matcher.Type)
			}
		}
	}

	return nil
}
```

The goal is visual grouping, not maximum whitespace. Several consecutive if statements are fine when they are related guards or naturally belong to the same logical group.
Do not introduce blank lines mechanically. Use them where they make the function's structure easier to scan.
