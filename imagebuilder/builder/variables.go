package builder

func (b *Builder) getVariableFunc(
	extraVariables0, extraVariables1 map[string]string) func(string) string {
	return func(varName string) string {
		if extraVariables0 != nil {
			if varValue, ok := extraVariables0[varName]; ok {
				return varValue
			}
		}
		if extraVariables1 != nil {
			if varValue, ok := extraVariables1[varName]; ok {
				return varValue
			}
		}
		return b.variables[varName]
	}
}
