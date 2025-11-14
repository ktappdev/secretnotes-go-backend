module.exports = {
  apps: [
    {
      name: "secretnotes",
      script: "git pull && go build && ./secretnotes-go-backend",
    },
  ],
};
