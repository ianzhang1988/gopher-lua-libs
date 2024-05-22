# golang lib net functionality

``` lua
local net = require("net")

-- @param1 domain
-- @param2 timeout
hosts, err = net.dnslookup("www.bing.com", 1)
if err ~= nil then
    print("err:" .. err)
end

for _, h in ipairs(hosts) do
    print("ip:" .. h)
end
```