server "138.197.4.16", user: "ralfbot"
set :deploy_to, "~/test"

set :application, "discord-dota-bot"
set :repo_url, "git@github.com:ralfizzle/discord-dota-bot.git"
set :pty, true

namespace :deploy do
	task :go_build do
		on roles :all do
			execute "cd #{release_path} && GOPATH=~/Projects/golang go build -o dota *.go"
			execute :sudo, "service dota stop"
			execute "cd #{release_path} && sqlite3 ~/.dota-config/dota.db < db.schema: true"
			execute :sudo, "service dota start"
		end
	end
end

after "deploy", "deploy:go_build"
