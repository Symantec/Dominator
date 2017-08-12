package builder

func (b *Builder) getVariableFunc(
	extraVariables map[string]string) func(string) string {
	return func(varName string) string {
		if extraVariables != nil {
			if varValue, ok := extraVariables[varName]; ok {
				return varValue
			}
		}
		return b.variables[varName]
	}
}
