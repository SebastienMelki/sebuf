package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SebastienMelki/sebuf/examples/nested-resources/api/proto/models"
	"github.com/SebastienMelki/sebuf/examples/nested-resources/api/proto/services"
)

// OrganizationService implements the OrganizationServiceServer interface.
type OrganizationService struct {
	orgs     map[string]*models.Organization
	teams    map[string]*models.Team    // key: orgId/teamId
	members  map[string]*models.Member  // key: orgId/teamId/memberId
	projects map[string]*models.Project // key: orgId/teamId/projectId
	nextID   int
}

// NewOrganizationService creates a new service with sample data.
func NewOrganizationService() *OrganizationService {
	svc := &OrganizationService{
		orgs:     make(map[string]*models.Organization),
		teams:    make(map[string]*models.Team),
		members:  make(map[string]*models.Member),
		projects: make(map[string]*models.Project),
		nextID:   1,
	}

	// Sample data
	now := time.Now().Unix()

	// Organizations
	svc.orgs["org-abc123"] = &models.Organization{
		Id:          "org-abc123",
		Name:        "Acme Corporation",
		Slug:        "acme-corp",
		Description: "Leading provider of innovative solutions",
		Website:     "https://acme.example.com",
		TeamCount:   2,
		MemberCount: 5,
		CreatedAt:   now,
	}
	svc.orgs["org-def456"] = &models.Organization{
		Id:          "org-def456",
		Name:        "TechStart Inc",
		Slug:        "techstart",
		Description: "Startup accelerator",
		Website:     "https://techstart.example.com",
		TeamCount:   1,
		MemberCount: 3,
		CreatedAt:   now,
	}

	// Teams
	svc.teams["org-abc123/team-xyz789"] = &models.Team{
		Id:           "team-xyz789",
		OrgId:        "org-abc123",
		Name:         "Engineering",
		Description:  "Core engineering team",
		Private:      false,
		MemberCount:  3,
		ProjectCount: 2,
		CreatedAt:    now,
	}
	svc.teams["org-abc123/team-mno456"] = &models.Team{
		Id:           "team-mno456",
		OrgId:        "org-abc123",
		Name:         "Design",
		Description:  "UI/UX design team",
		Private:      false,
		MemberCount:  2,
		ProjectCount: 1,
		CreatedAt:    now,
	}

	// Members
	svc.members["org-abc123/team-xyz789/member-456"] = &models.Member{
		Id:        "member-456",
		UserId:    "user-john123",
		UserName:  "John Doe",
		UserEmail: "john@example.com",
		TeamId:    "team-xyz789",
		OrgId:     "org-abc123",
		Role:      models.MemberRole_MEMBER_ROLE_OWNER,
		JoinedAt:  now,
	}
	svc.members["org-abc123/team-xyz789/member-457"] = &models.Member{
		Id:        "member-457",
		UserId:    "user-jane456",
		UserName:  "Jane Smith",
		UserEmail: "jane@example.com",
		TeamId:    "team-xyz789",
		OrgId:     "org-abc123",
		Role:      models.MemberRole_MEMBER_ROLE_ADMIN,
		JoinedAt:  now,
	}

	// Projects
	svc.projects["org-abc123/team-xyz789/proj-def456"] = &models.Project{
		Id:          "proj-def456",
		TeamId:      "team-xyz789",
		OrgId:       "org-abc123",
		Name:        "API Redesign",
		Description: "Redesign the core API for v2",
		Status:      models.ProjectStatus_PROJECT_STATUS_ACTIVE,
		Tags:        []string{"api", "high-priority"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	svc.projects["org-abc123/team-xyz789/proj-ghi789"] = &models.Project{
		Id:          "proj-ghi789",
		TeamId:      "team-xyz789",
		OrgId:       "org-abc123",
		Name:        "Mobile App",
		Description: "New mobile application",
		Status:      models.ProjectStatus_PROJECT_STATUS_PLANNING,
		Tags:        []string{"mobile", "ios", "android"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return svc
}

// === Organization Methods ===

func (s *OrganizationService) ListOrganizations(ctx context.Context, req *models.ListOrganizationsRequest) (*models.ListOrganizationsResponse, error) {
	var orgs []*models.Organization
	for _, org := range s.orgs {
		orgs = append(orgs, org)
	}
	return &models.ListOrganizationsResponse{
		Organizations: orgs,
		TotalCount:    int32(len(orgs)),
		Page:          1,
		TotalPages:    1,
	}, nil
}

func (s *OrganizationService) GetOrganization(ctx context.Context, req *models.GetOrganizationRequest) (*models.Organization, error) {
	org, exists := s.orgs[req.OrgId]
	if !exists {
		return nil, fmt.Errorf("organization not found: %s", req.OrgId)
	}
	return org, nil
}

func (s *OrganizationService) CreateOrganization(ctx context.Context, req *models.CreateOrganizationRequest) (*models.Organization, error) {
	org := &models.Organization{
		Id:          fmt.Sprintf("org-%d", s.nextID),
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Website:     req.Website,
		CreatedAt:   time.Now().Unix(),
	}
	s.nextID++
	s.orgs[org.Id] = org
	return org, nil
}

// === Team Methods ===

func (s *OrganizationService) ListTeams(ctx context.Context, req *models.ListTeamsRequest) (*models.ListTeamsResponse, error) {
	var teams []*models.Team
	for key, team := range s.teams {
		if team.OrgId == req.OrgId {
			if req.IncludePrivate || !team.Private {
				teams = append(teams, team)
			}
		}
		_ = key
	}
	return &models.ListTeamsResponse{
		Teams:      teams,
		TotalCount: int32(len(teams)),
		Page:       1,
		TotalPages: 1,
	}, nil
}

func (s *OrganizationService) GetTeam(ctx context.Context, req *models.GetTeamRequest) (*models.Team, error) {
	key := fmt.Sprintf("%s/%s", req.OrgId, req.TeamId)
	team, exists := s.teams[key]
	if !exists {
		return nil, fmt.Errorf("team not found: %s in org %s", req.TeamId, req.OrgId)
	}
	return team, nil
}

func (s *OrganizationService) CreateTeam(ctx context.Context, req *models.CreateTeamRequest) (*models.Team, error) {
	team := &models.Team{
		Id:          fmt.Sprintf("team-%d", s.nextID),
		OrgId:       req.OrgId,
		Name:        req.Name,
		Description: req.Description,
		Private:     req.Private,
		CreatedAt:   time.Now().Unix(),
	}
	s.nextID++
	key := fmt.Sprintf("%s/%s", req.OrgId, team.Id)
	s.teams[key] = team
	return team, nil
}

// === Member Methods ===

func (s *OrganizationService) ListTeamMembers(ctx context.Context, req *models.ListTeamMembersRequest) (*models.ListTeamMembersResponse, error) {
	var members []*models.Member
	prefix := fmt.Sprintf("%s/%s/", req.OrgId, req.TeamId)
	for key, member := range s.members {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			members = append(members, member)
		}
	}
	return &models.ListTeamMembersResponse{
		Members:    members,
		TotalCount: int32(len(members)),
		Page:       1,
		TotalPages: 1,
	}, nil
}

func (s *OrganizationService) GetTeamMember(ctx context.Context, req *models.GetTeamMemberRequest) (*models.Member, error) {
	key := fmt.Sprintf("%s/%s/%s", req.OrgId, req.TeamId, req.MemberId)
	member, exists := s.members[key]
	if !exists {
		return nil, fmt.Errorf("member not found: %s in team %s, org %s", req.MemberId, req.TeamId, req.OrgId)
	}
	return member, nil
}

func (s *OrganizationService) AddTeamMember(ctx context.Context, req *models.AddTeamMemberRequest) (*models.Member, error) {
	member := &models.Member{
		Id:        fmt.Sprintf("member-%d", s.nextID),
		UserId:    req.UserId,
		UserName:  "New User",
		UserEmail: fmt.Sprintf("%s@example.com", req.UserId),
		TeamId:    req.TeamId,
		OrgId:     req.OrgId,
		Role:      req.Role,
		JoinedAt:  time.Now().Unix(),
	}
	s.nextID++
	key := fmt.Sprintf("%s/%s/%s", req.OrgId, req.TeamId, member.Id)
	s.members[key] = member
	return member, nil
}

func (s *OrganizationService) RemoveTeamMember(ctx context.Context, req *models.RemoveTeamMemberRequest) (*models.RemoveTeamMemberResponse, error) {
	key := fmt.Sprintf("%s/%s/%s", req.OrgId, req.TeamId, req.MemberId)
	if _, exists := s.members[key]; !exists {
		return nil, fmt.Errorf("member not found: %s", req.MemberId)
	}
	delete(s.members, key)
	return &models.RemoveTeamMemberResponse{
		Success: true,
		Message: fmt.Sprintf("Member %s removed from team %s", req.MemberId, req.TeamId),
	}, nil
}

// === Project Methods ===

func (s *OrganizationService) ListProjects(ctx context.Context, req *models.ListProjectsRequest) (*models.ListProjectsResponse, error) {
	var projects []*models.Project
	prefix := fmt.Sprintf("%s/%s/", req.OrgId, req.TeamId)
	for key, project := range s.projects {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			projects = append(projects, project)
		}
	}
	return &models.ListProjectsResponse{
		Projects:   projects,
		TotalCount: int32(len(projects)),
		Page:       1,
		TotalPages: 1,
	}, nil
}

func (s *OrganizationService) GetProject(ctx context.Context, req *models.GetProjectRequest) (*models.Project, error) {
	key := fmt.Sprintf("%s/%s/%s", req.OrgId, req.TeamId, req.ProjectId)
	project, exists := s.projects[key]
	if !exists {
		return nil, fmt.Errorf("project not found: %s in team %s, org %s", req.ProjectId, req.TeamId, req.OrgId)
	}
	return project, nil
}

func (s *OrganizationService) CreateProject(ctx context.Context, req *models.CreateProjectRequest) (*models.Project, error) {
	now := time.Now().Unix()
	project := &models.Project{
		Id:          fmt.Sprintf("proj-%d", s.nextID),
		TeamId:      req.TeamId,
		OrgId:       req.OrgId,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		Tags:        req.Tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.nextID++
	key := fmt.Sprintf("%s/%s/%s", req.OrgId, req.TeamId, project.Id)
	s.projects[key] = project
	return project, nil
}

func main() {
	// Use our custom service implementation
	service := NewOrganizationService()
	mux := http.NewServeMux()

	// Register the HTTP handlers (generated by protoc-gen-go-http)
	if err := services.RegisterOrganizationServiceServer(service, services.WithMux(mux)); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Nested Resources API Server starting on :8080")
	fmt.Println("")
	fmt.Println("This example demonstrates complex nested resource hierarchies")
	fmt.Println("with multiple path parameters (GitHub-like org/team/member pattern).")
	fmt.Println("")
	fmt.Println("Resource Hierarchy:")
	fmt.Println("  Organization (1 path param)")
	fmt.Println("    -> Team (2 path params)")
	fmt.Println("       -> Member (3 path params)")
	fmt.Println("       -> Project (3 path params)")
	fmt.Println("")
	fmt.Println("Endpoints:")
	fmt.Println("")
	fmt.Println("  Organizations:")
	fmt.Println("    GET  /api/v1/orgs                    - List all organizations")
	fmt.Println("    GET  /api/v1/orgs/{org_id}           - Get organization")
	fmt.Println("    POST /api/v1/orgs                    - Create organization")
	fmt.Println("")
	fmt.Println("  Teams (nested under org):")
	fmt.Println("    GET  /api/v1/orgs/{org_id}/teams              - List teams")
	fmt.Println("    GET  /api/v1/orgs/{org_id}/teams/{team_id}    - Get team")
	fmt.Println("    POST /api/v1/orgs/{org_id}/teams              - Create team")
	fmt.Println("")
	fmt.Println("  Members (nested under team):")
	fmt.Println("    GET    /api/v1/orgs/{org_id}/teams/{team_id}/members              - List members")
	fmt.Println("    GET    /api/v1/orgs/{org_id}/teams/{team_id}/members/{member_id}  - Get member")
	fmt.Println("    POST   /api/v1/orgs/{org_id}/teams/{team_id}/members              - Add member")
	fmt.Println("    DELETE /api/v1/orgs/{org_id}/teams/{team_id}/members/{member_id}  - Remove member")
	fmt.Println("")
	fmt.Println("  Projects (nested under team):")
	fmt.Println("    GET  /api/v1/orgs/{org_id}/teams/{team_id}/projects               - List projects")
	fmt.Println("    GET  /api/v1/orgs/{org_id}/teams/{team_id}/projects/{project_id}  - Get project")
	fmt.Println("    POST /api/v1/orgs/{org_id}/teams/{team_id}/projects               - Create project")
	fmt.Println("")
	fmt.Println("Example: Get a member (3 path parameters):")
	fmt.Println("  curl -X GET http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789/members/member-456 \\")
	fmt.Println("    -H 'Authorization: Bearer test-token'")
	fmt.Println("")

	log.Fatal(http.ListenAndServe(":8080", mux))
}
