module.exports = {
  apps: [
    {
      name: "secretnotes-backend",
      script: "git pull && GOTOOLCHAIN=go1.26.5 go build && ./secretnotes-go-backend",
    },
  ],
};
