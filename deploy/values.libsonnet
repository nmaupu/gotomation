local default_values = importstr 'values-default.yaml';
local values = importstr 'values.yaml';

std.mergePatch(std.parseYaml(default_values), std.parseYaml(values))
