module.exports = {
  apps: [
    {
      name: "secretnotes-backend",
      script: "git pull && go build && ./secretnotes-go-backend",
    },
  ],
};
