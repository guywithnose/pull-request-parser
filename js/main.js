(function(window, $, Promise) {
  var MIN_APPROVALS = 2;

  var dataTable = $('#approved-prs').DataTable({
    paging: false,
    info: false,
    search: {
      regex: true
    },
    classes: {
      sFilterInput: 'form-control'
    },
    language: {
      search: '<div class="col-xs-2"><label class="control-label">Search:</label></div><div class="col-xs-8">_INPUT_</div>'
    },
    columns: [
      {'visible': false},
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null,
      null
    ]
  });

  var GhApi = function(apiUrl, token) {
    this.apiUrl = apiUrl;
    var ajaxOptions = {
      dataType: 'json',
      cache: false,
      headers: {Authorization: 'token ' + token, Accept: 'application/vnd.github.black-cat-preview.full+json'}
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

  GhApi.prototype.getBranchCommit = function(repoPath, branch) {
    return this.ajax(this.apiUrl + '/repos/' + repoPath + '/commits/' + (branch || 'master')).then(function(commit) {
      commit.branch = branch;
      return commit;
    });
  };

  GhApi.prototype.getReviews = function(repoPath, prNum) {
    // We have to catch this once since not all versions of github support it
    return this.ajax(this.apiUrl + '/repos/' + repoPath + '/pulls/' + prNum + '/reviews').catch(function() {return [];});
  };

  GhApi.prototype.getOrganizationRepos = function(organization) {
    return this.ajax(this.apiUrl + '/orgs/' + organization + '/repos');
  };

  GhApi.prototype.getPullLabels = function(repoPath, prId) {
    return this.ajax(this.apiUrl + '/repos/' + repoPath + '/issues/' + prId + '/labels');
  };

  GhApi.prototype.getUserRepos = function(organization) {
    return this.ajax(this.apiUrl + '/users/' + organization + '/repos');
  };

  GhApi.prototype.getRepoData = function(repoPath) {
    return this.ajax(this.apiUrl + '/repos/' + repoPath);
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
  };

  GhApi.prototype.getRepoPull = function(repoPath, prNum) {
    var self = this;

    return this.ajax(this.apiUrl + '/repos/' + repoPath + '/pulls/' + prNum)
      .then(function(pull) {
        return self.getPullDetails(pull).then(function(details) {
          return $.extend(pull, details);
        });
      });
  };

  GhApi.prototype.getPullDetails = function(pullRequest) {
    return Promise.props({
      comments: this.ajax(pullRequest.comments_url),
      review_comments: this.ajax(pullRequest.review_comments_url),
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
  };

  var LS = function(namespace) {
    this.keyOf = function(name) {
      return name + ':' + namespace;
    };
  };

  LS.prototype.getAccessToken = function() {
    return window.localStorage[this.keyOf('github_access_token')];
  };

  LS.prototype.setAccessToken = function(accessToken) {
    return window.localStorage[this.keyOf('github_access_token')] = accessToken;
  };

  LS.prototype.unsetAccessToken = function() {
    delete window.localStorage[this.keyOf('github_access_token')];
  };

  LS.prototype.getRepoPaths = function() {
    var repos = window.localStorage[this.keyOf('repos')];

    return repos ? JSON.parse(repos) : [];
  };

  LS.prototype.addRepoPath = function(repoPath) {
    var repoPaths = this.getRepoPaths();
    if (repoPaths.indexOf(repoPath) === -1) {
      repoPaths.push(repoPath);
      localStorage[this.keyOf('repos')] = JSON.stringify(repoPaths);
    }
  };

  LS.prototype.removeRepoPath = function(repoPath) {
    var repoPaths = this.getRepoPaths();
    var index = repoPaths.indexOf(repoPath);
    if (index !== -1) {
      repoPaths = repoPaths.splice(0, index).concat(repoPaths.splice(index + 1));
      localStorage[this.keyOf('repos')] = JSON.stringify(repoPaths);
    }
  };

  function updateOptionBoxes(repoPaths) {
    repoPaths.sort();
    var leftOptionGroup = $('#repoPathOptionsLeft').html('');
    var rightOptionGroup = $('#repoPathOptionsRight').html('');
    var leftOptions, rightOptions;
    if (repoPaths.length <= 3) {
      leftOptions = repoPaths;
      rightOptions = [];
    } else {
      var middleIndex = Math.ceil(repoPaths.length / 2);
      leftOptions = repoPaths.slice(0, middleIndex);
      rightOptions = repoPaths.slice(middleIndex);
    }

    buildOptionBox(leftOptionGroup, leftOptions);
    buildOptionBox(rightOptionGroup, rightOptions);
  }

  function buildOptionBox(box, repoPaths) {
    $.each(repoPaths, function(index, repoPath) {
      box.append(
        '<a class="list-group-item list-group-item-info btn-danger btn-link" data-repo-path="' + repoPath + '"><span class="repoPathOption">' +
        repoPath + '</span><span class="badge glyphicon glyphicon-remove btn btn-danger"> </span></a>'
      );
    });
  }

  function parsePullRequests(ghApi, repo) {
    var repoPath = repo.full_name;

    Promise.join(
        ghApi.getUser(),
        ghApi.getRepoPulls(repoPath).then(function(pulls) {
          var baseBranches = [];
          var baseBranchPromises = [];
          var labelPromises = [];
          for (var i in pulls) {
            var baseBranch = pulls[i].base.ref;
            labelPromises.push(ghApi.getPullLabels(repoPath, pulls[i].number));
            if (baseBranches.indexOf(baseBranch) === -1) {
              baseBranches.push(baseBranch);
              baseBranchPromises.push(ghApi.getBranchCommit(repoPath, baseBranch));
            }
          }

          return Promise.join(
            Promise.all(baseBranchPromises),
            Promise.all(labelPromises),
            function(baseBranchCommits, labels) {
              baseBranchCommitsObject = {};
              for (var i in baseBranchCommits) {
                baseBranchCommitsObject[baseBranchCommits[i].branch] = baseBranchCommits[i];
              }

              for (var i in labels) {
                pulls[i].labels = labels[i];
              }

              return {pulls: pulls, baseBranchCommits: baseBranchCommitsObject};
            }
          );
        }),
        function(user, pullData) {
          parseAllPullRequests(ghApi, user, pullData.baseBranchCommits, pullData.pulls);
        }
    );
  }

  function parseRepos(ghApi, specs) {
    updateEmptyTableText('Loading data <span class="glyphicon glyphicon-refresh glyphicon-refresh-animate"></span>');
    $.each(specs, function(index, spec) {
      if (spec.indexOf('/') === -1) {
        ghApi.getOrganizationRepos(spec).catch(function() {
          return ghApi.getUserRepos(spec);
        }).each(function(repo) {
          parsePullRequests(ghApi, repo);
        }).catch(function() {
          updateEmptyTableText('No pull requests found');
        });
      } else {
        ghApi.getRepoData(spec).then(function(repo) {
          parsePullRequests(ghApi, repo);
        }).catch(function() {
          updateEmptyTableText('No pull requests found');
        });
      }
    });
  }

  function updateEmptyTableText(text) {
    dataTable.settings()[0].oLanguage.sEmptyTable = text;
    dataTable.draw();
  }

  function parseAllPullRequests(ghApi, user, baseBranchCommits, pulls) {
    var username = user.login;
    for (var i in pulls) {
      getReviewsAndParsePullRequest(ghApi, username, baseBranchCommits[pulls[i].base.ref].sha, pulls[i]);
    }
  }

  function getReviewsAndParsePullRequest(ghApi, username, sha, pullRequest) {
    ghApi.getReviews(pullRequest.base.repo.owner.login + '/' + pullRequest.base.repo.name, pullRequest.number).then(function(reviews) {
      parsePullRequest(username, sha, pullRequest, reviews);
    });
  }

  function refreshPr(ghApi, repoPath, prNum) {
    ghApi.getRepoData(repoPath).then(function(repo) {
      Promise.join(
          ghApi.getUser(),
          ghApi.getRepoPull(repoPath, prNum).then(function(pull) {
            return ghApi.getBranchCommit(repoPath, pull.base.ref).then(function(commit) {
              return {commit: commit, pull: pull};
            });
          }),
          ghApi.getPullLabels(repoPath, prNum),
          ghApi.getReviews(repoPath, prNum),
          function(user, pullData, labels, reviews) {
            pullData.pull.labels = labels;
            parsePullRequest(user.login, pullData.commit.sha, pullData.pull, reviews);
          }
      );
    });
  }

  function parsePullRequest(username, headCommit, pullRequest, reviews) {
    pullRequest.iAmOwner = pullRequest.user.login == username;
    pullRequest.approvals = approvingComments(pullRequest.comments.concat(pullRequest.review_comments), reviews);
    pullRequest.numApprovals = Object.keys(pullRequest.approvals).length;
    pullRequest.approved = pullRequest.numApprovals >= MIN_APPROVALS;
    pullRequest.iHaveApproved = !!pullRequest.approvals[username];
    pullRequest.isRebased = ancestryContains(pullRequest.commits, headCommit);
    pullRequest.rebasedText = pullRequest.isRebased ? 'Y' : 'N';
    pullRequest.state = getState(pullRequest.statuses);
    pullRequest.needsMyApproval = !pullRequest.iHaveApproved && !pullRequest.iAmOwner ? 'Y' : 'N';
    pullRequest.labelHtml = buildLabels(pullRequest.labels);

    dataTable.row.add([
      JSON.stringify(pullRequest),
      '<a href="' + pullRequest.base.repo.html_url + '" target="_blank">' + pullRequest.base.repo.full_name + '</a>',
      '<a href="' + pullRequest.html_url + '" target="_blank">' + pullRequest.number + '</a>',
      pullRequest.user.login,
      pullRequest.head.ref,
      pullRequest.base.ref,
      "<div data-toggle='popover' data-html='true' data-trigger='hover' data-content='" + approvalHtml(pullRequest) + "'>" +
        pullRequest.numApprovals + '</td>',
      pullRequest.rebasedText,
      pullRequest.state,
      pullRequest.needsMyApproval,
      pullRequest.labelHtml,
      '<button class="refresh">Refresh</button>'
    ]).draw().nodes().to$().addClass(rowClass(pullRequest)).data(
      {prNum: pullRequest.number, repoPath: pullRequest.base.repo.full_name}
    ).find('td a').each(function(index, element) {
      $(element).parent().addClass('pointer');
    });
    $('[data-toggle="popover"]').popover();
  };

  function buildLabels(labels) {
    var labelHtml = '';
    for (var i in labels) {
      labelHtml += '<span title="' + labels[i].name +
        '" style="margin: 0 2px; height: 22px; overflow: hidden; width: 10px; text-align: center; line-height: 0.9; color: white; ' +
        'font-size: 8px; display: inline-block; background-color: #' + labels[i].color +
        ';">' + $.trim(labels[i].name.replace(/\b([a-zA-Z])[a-zA-Z]*([^a-zA-Z]+|$)/g, '$1 ')).toUpperCase() + '</span>';
    }

    return labelHtml;
  }

  function getState(statuses) {
    if (statuses.length == 0) {
      return 'none';
    }

    var contexts = {};
    for (var i in statuses) {
      if (!contexts[statuses[i].context]) {
        contexts[statuses[i].context] = '?';
      }

      if (statuses[i].state === 'success') {
        contexts[statuses[i].context] = 'Y';
      }

      if (statuses[i].state === 'failure') {
        contexts[statuses[i].context] = 'N';
      }
    }

    var html = [];
    for (var i in contexts) {
      html.push('<span title="' + i + '">' + contexts[i] + '</span>');
    }

    return html.join('/');
  }

  /*
   * Returns the users that have a comment containing :+1: or LGTM.
   */
  function approvingComments(comments, reviews) {
    var result = {};
    for (var i in comments) {
      if (isApproval(comments[i]) && $.inArray(comments[i].user.login, result) === -1) {
        if (!result[comments[i].user.login]) {
          result[comments[i].user.login] = [];
        }

        result[comments[i].user.login].push(comments[i].body_html);
      }
    }

    for (var i in reviews) {
      if (reviews[i].state === 'APPROVED') {
        if (!result[reviews[i].user.login]) {
          result[reviews[i].user.login] = [];
        }

        result[reviews[i].user.login].push(reviews[i].body_html);
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

  function approvalHtml(pullRequest) {
    var comments = [];
    for (var commentor in pullRequest.approvals) {
      for (var i in pullRequest.approvals[commentor]) {
        comments.push(commentor + ': ' + pullRequest.approvals[commentor][i]);
      }
    }

    return comments.join('<BR><BR>');
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
        $('#pr-data').show();
      }).catch(function() {
        ls.unsetAccessToken();
        $('#getAccessToken').show();
      });
    } else {
      $('#getAccessToken').show();
    }

    updateOptionBoxes(ls.getRepoPaths());

    $('#repoPathOptions').on('click', 'a span.badge', function(event) {
      ls.removeRepoPath($(this).parents('.list-group-item').data().repoPath);
      updateOptionBoxes(ls.getRepoPaths());
      event.stopPropagation();
    });

    $('#repoPathOptions').on('click', 'a', function() {
      dataTable.clear().draw();
      parseRepos(ghApi, [$(this).data().repoPath]);
    });

    $('#approved-prs').on('click', 'td', function() {
      var link = $(this).find('a');
      if (link.length === 1 && link.attr('href')) {
        window.open(link.attr('href'), '_blank');
      }
    });

    $('#approved-prs').on('click', 'td a', function(event) {
      event.stopPropagation();
    });

    $('#saveAccessToken').click(function() {
      ghApi = new GhApi(apiUrl, $('#accessToken').val());
      ghApi.getUser().then(function() {
        ls.setAccessToken($('#accessToken').val());
        $('#getAccessToken').hide();
        $('#pickRepo').show();
        $('#pr-data').show();
      }).catch(function() {
        alert('It appears that access token is invalid');
      });
    });

    $('#parsePullRequests').click(function() {
      var repoPaths = $('#repoPath').val().split('\n');

      $.each(repoPaths, function(index, repoPath) {
        ls.addRepoPath(repoPath);
      });
      updateOptionBoxes(ls.getRepoPaths());

      dataTable.clear().draw();
      parseRepos(ghApi, repoPaths);
    });

    $('#checkAllRepos').click(function() {
      dataTable.clear().draw();
      parseRepos(ghApi, ls.getRepoPaths());
    });

    $('#approved-prs').on('click', '.refresh', function() {
      var row = $(this).parents('tr');
      var repoPath = row.data('repoPath');
      var prNum = row.data('prNum');
      dataTable.row($(this).parents('tr')).remove().draw();
      refreshPr(ghApi, repoPath, prNum);
    });
  };

  window.PullRequestParser = PullRequestParser;
}(window, jQuery, Promise));
