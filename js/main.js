(function(window, $, Promise) {
  var MIN_APPROVALS = 2;

  var GhApi = function(apiUrl, token) {
    this.apiUrl = apiUrl;
    var ajaxOptions = {
      dataType: "json",
      cache: false,
      headers: {Authorization: 'token ' + token}
    };

    this.ajax = function(url) {
      return Promise.resolve(
        $.ajax(url, ajaxOptions).then(function(data, status, xhr) {
          var link = this.parseLinkHeader(xhr.getResponseHeader('Link'));

          if (typeof link.next === 'undefined') {
            return data;
          }

          return this.ajax(link.next).then(function(next) {
            return data.concat(next);
          });
        }.bind(this))
      );
    };
  };

  GhApi.prototype.getUser = function() {
    return this.ajax(this.apiUrl + '/user');
  };

  GhApi.prototype.getRepoCommits = function(repoPath) {
    return this.ajax(this.apiUrl + '/repos/' + repoPath + '/commits/master');
  }

  GhApi.prototype.getOrganizationRepos = function(organization) {
    return this.ajax(this.apiUrl + '/orgs/' + organization + '/repos');
  };

  GhApi.prototype.getRepoPulls = function(repoPath) {
    var self = this;

    return Promise.map(
      this.ajax(this.apiUrl + '/repos/' + repoPath + '/pulls'),
      function(pull) {
        return self.getPullDetails(pull).then(function(details) {
          return $.extend(pull, details);
        });
      }
    );
  }

  GhApi.prototype.getRepoPull = function(repoPath, prNum) {
    var self = this;

    return this.ajax(this.apiUrl + '/repos/' + repoPath + '/pulls/' + prNum)
      .then(function(pull) {
        return self.getPullDetails(pull).then(function(details) {
          return $.extend(pull, details);
        });
      });
  }

  GhApi.prototype.getPullDetails = function(pullRequest) {
    return Promise.props({
      comments: this.ajax(pullRequest.comments_url),
      commits: this.ajax(pullRequest.commits_url || (pullRequest.url + '/commits')),
      statuses: this.ajax(pullRequest.statuses_url || pullRequest.base.repo.statuses_url.replace('{sha}', pullRequest.head.sha))
    });
  };

  GhApi.prototype.parseLinkHeader = function(header) {
    var result = {};
    if (typeof header !== 'string') {
      return result;
    }

    $.each(header.split(','), function(i, link) {
      var sections = link.split(';');
      var url = sections[0].replace(/<(.*)>/, '$1').trim();
      var name = sections[1].replace(/rel="(.*)"/, '$1').trim();
      result[name] = url;
    });

    return result;
  } ;

  var LS = function(namespace) {
    this.keyOf = function(name) {
      return name + ':' + namespace;
    }
  }

  LS.prototype.getAccessToken = function() {
    return window.localStorage[this.keyOf('github_access_token')];
  }

  LS.prototype.setAccessToken = function(accessToken) {
    return window.localStorage[this.keyOf('github_access_token')] = accessToken;
  }

  LS.prototype.unsetAccessToken = function() {
    delete window.localStorage[this.keyOf('github_access_token')];
  }

  LS.prototype.getRepoPaths = function() {
    var repos = window.localStorage[this.keyOf('repos')];

    return repos ? JSON.parse(repos) : [];
  }

  LS.prototype.addRepoPath = function(repoPath) {
    var repoPaths = this.getRepoPaths();
    if (repoPaths.indexOf(repoPath) === -1) {
      repoPaths.push(repoPath);
      localStorage[this.keyOf('repos')] = JSON.stringify(repoPaths);
    }
  }

  function updateSelectBoxes(repoPaths) {
    $('#repoPathSelect').html('<option></option>');
    if (repoPaths) {
      for (var i in repoPaths) {
          $('#repoPathSelect').append('<option>' + repoPaths[i] + '</option>');
      }

      $('#repoPathSelect').show();
    }
  }

  function parsePullRequests(ghApi, repoPath) {
    Promise.join(
        ghApi.getUser(),
        ghApi.getRepoCommits(repoPath),
        ghApi.getRepoPulls(repoPath),
        function(user, commits, pulls) {
          parseAllPullRequests(user, commits, pulls);
        }
    );
  }

  function parseRepos(ghApi, specs) {
    $.each(specs, function(index, spec) {
      if (spec.indexOf('/') === -1) {
        ghApi.getOrganizationRepos(spec).each(function(repo) {
          parsePullRequests(ghApi, repo.full_name);
        });
      } else {
        parsePullRequests(ghApi, spec);
      }
    });
  }

  function parseAllPullRequests(user, commit, pulls) {
    var username = user.login;
    var headCommit = commit.sha;
    for (var i in pulls) {
      parsePullRequest(username, headCommit, pulls[i]);
    }
  }

  function refreshPr(ghApi, repoPath, prNum) {
    Promise.join(
        ghApi.getUser(),
        ghApi.getRepoCommits(repoPath),
        ghApi.getRepoPull(repoPath, prNum),
        function(user, commit, pull) {
          parsePullRequest(user.login, commit.sha, pull);
        }
    );
  }

  function parsePullRequest(username, headCommit, pullRequest) {
    pullRequest.iAmOwner = pullRequest.user.login == username;
    pullRequest.approvals = approvingComments(pullRequest.comments);
    pullRequest.numApprovals = Object.keys(pullRequest.approvals).length;
    pullRequest.approved = pullRequest.numApprovals >= MIN_APPROVALS;
    pullRequest.iHaveApproved = !!pullRequest.approvals[username];
    pullRequest.isRebased = ancestryContains(pullRequest.commits, headCommit);
    pullRequest.rebasedText = pullRequest.isRebased ? 'Y' : 'N';
    var state = getState(pullRequest.statuses);
    pullRequest.state = state == 'success' ? 'Y' : state == 'none' || state == 'pending' ? '?' : 'N';
    pullRequest.needsMyApproval = !pullRequest.iHaveApproved && !pullRequest.iAmOwner ? 'Y' : 'N';

    buildHtml(pullRequest);
  };

  function buildHtml(pullRequest) {
      var html = buildRow(pullRequest);

      if (pullRequest.approved) {
        $('#approved-prs tbody').prepend(html);
      } else {
        $('#approved-prs tbody').append(html);
      }
  }

  function getState(statuses) {
    if (statuses.length == 0) {
      return 'none';
    }

    return statuses[0].state;
  }

  /*
   * Returns the users that have a comment containing :+1: or LGTM.
   */
  function approvingComments(comments) {
    var result = {};
    for (var i in comments) {
      if (isApproval(comments[i]) && $.inArray(comments[i].user.login, result) === -1) {
        if (!result[comments[i].user.login]) {
          result[comments[i].user.login] = [];
        }

        result[comments[i].user.login].push(comments[i].body);
      }
    }

    return result;
  }

  function isApproval(comment) {
    return comment.body.search(':\\+1:') != -1 ||
      comment.body.search(':thumbsup:') != -1 ||
      comment.body.search('LGTM') != -1;
  }

  /*
   * Searches through the commits and checks to see if any of them contain the requested commit hash
   */
  function ancestryContains(commits, commitHash) {
    for (var i in commits) {
      for (var j in commits[i].parents) {
        var parent = commits[i].parents[j];
        if (parent.sha == commitHash) {
          return true;
        }
      }
    }

    return false;
  }

  function buildRow(pullRequest) {
    var row = '<td>' + pullRequest.base.repo.full_name + '</td>' +
      '<td><a href="' + pullRequest.html_url + '" target="_blank">' + pullRequest.number + '</a></td>' +
      '<td>' + pullRequest.user.login + '</td>' +
      '<td>' + pullRequest.head.ref + '</td>' +
      '<td title="' + approvalTitle(pullRequest) + '">' + pullRequest.numApprovals + '</td>' +
      '<td>' + pullRequest.rebasedText + '</td>' +
      '<td>' + pullRequest.state + '</td>' +
      '<td>' + pullRequest.needsMyApproval + '</td>' +
      '<td><button class="refresh">Refresh</button></td>';

    return '<tr data-pr-num="' + pullRequest.number + '" data-repo-path="' + pullRequest.base.repo.full_name + '" class="' + rowClass(pullRequest) + '" data-link="' + pullRequest.html_url + '">' + row + + '</tr>';
  }

  function approvalTitle(pullRequest) {
    var title = '';
    for (var commentor in pullRequest.approvals) {
      for (var i in pullRequest.approvals[commentor]) {
        title += commentor + ': ' + pullRequest.approvals[commentor][i] + '\n';
      }
    }

    return title;
  }

  function rowClass(pullRequest) {
    if (pullRequest.approved && pullRequest.isRebased) {
      return 'success';
    }
    
    if (!pullRequest.iHaveApproved && !pullRequest.iAmOwner) {
      return 'info';
    }
    
    if (pullRequest.iAmOwner && !pullRequest.isRebased) {
      return 'warning';
    }

    if (pullRequest.state == 'failure') {
      return 'danger';
    }

    return '';
  }

  var PullRequestParser = function(options) {
    options = options || {};

    var apiUrl = options.apiUrl || 'https://api.github.com';
    MIN_APPROVALS = options.minApprovals || MIN_APPROVALS;

    var ls = new LS(apiUrl);
    var ghApi;

    if (ls.getAccessToken()) {
      ghApi = new GhApi(apiUrl, ls.getAccessToken());
      ghApi.getUser().then(function() {
        $('#pickRepo').show();
      }).catch(function() {
        ls.unsetAccessToken();
        $('#getAccessToken').show();
      });
    } else {
      $('#getAccessToken').show();
    }

    updateSelectBoxes(ls.getRepoPaths());

    $('#saveAccessToken').click(function() {
      ghApi = new GhApi(apiUrl, $('#accessToken').val());
      ghApi.getUser().then(function() {
        ls.setAccessToken($('#accessToken').val());
        $('#getAccessToken').hide();
        $('#pickRepo').show();
      }).catch(function() {
        alert('It appears that access token is invalid');
      });
    });

    $('#parsePullRequests').click(function() {
      var repoPaths = $('#repoPath').val().split('\n');

      $.each(repoPaths, function(index, repoPath) {
        ls.addRepoPath(repoPath);
      });
      updateSelectBoxes(ls.getRepoPaths());

      $('#approved-prs tbody').html('');
      parseRepos(ghApi, repoPaths);
    });

    $('#checkAllRepos').click(function() {
      $('#approved-prs tbody').html('');
      parseRepos(ghApi, ls.getRepoPaths());
    });

    $('#repoPathSelect').change(function(){
      $('#repoPath').val($(this).val());
    });

    $('#approved-prs').on('click', '.refresh', function() {
      var row = $(this).parents('tr');
      var repoPath = row.data('repoPath');
      var prNum = row.data('prNum');
      $(this).parent().parent().remove();
      refreshPr(ghApi, repoPath, prNum);
    });
  };

  window.PullRequestParser = PullRequestParser;
}(window, jQuery, Promise))
