local socket_ok, socket = pcall(require, "socket")
if not socket_ok then
    print("Error: LuaSocket module not found")
    print("Install LuaSocket:")
    print("  - macOS: brew install lua-socket")
    print("  - Ubuntu/Debian: apt-get install lua-socket (or lua5.4-socket for Lua 5.4)")
    print("  - Fedora: dnf install lua-socket")
    print("  - Generic: luarocks install luasocket")
    print()
    print("Note: Module must be installed for Lua " .. _VERSION)
    os.exit(1)
end

local server = socket.tcp()
server:bind("*", 8080)
server:listen(1)

print("HTTP Server listening on http://localhost:8080")
print("Press Ctrl+C to stop")

while true do
    local client = server:accept()

    local request, err = client:receive()
    if request then
        local method = request:match("(%w+) ")

        if method == "GET" then
            local response = '{"status":"ok","message":"Hello from Lua"}'
            client:send("HTTP/1.1 200 OK\r\n")
            client:send("Content-Type: application/json\r\n")
            client:send("Content-Length: " .. #response .. "\r\n")
            client:send("\r\n")
            client:send(response)
        else
            local response = '{"error":"Only GET method allowed"}'
            client:send("HTTP/1.1 405 Method Not Allowed\r\n")
            client:send("Content-Type: application/json\r\n")
            client:send("Content-Length: " .. #response .. "\r\n")
            client:send("\r\n")
            client:send(response)
        end
    end

    client:close()
end
