module.exports = {
    apps: [
      {
        name: 'secure-citizen-backend',
        script: "git pull && go build && ./secretnotes serve",
      },
    ],
  };
  